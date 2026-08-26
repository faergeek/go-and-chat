package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/faergeek/go-and-chat/internal/proto"
	"github.com/faergeek/go-and-chat/internal/terminal"
)

func Start(address string) error {
	if !terminal.IsTerminal() {
		return errors.New("Non-interactive use is not supported")
	}

	term, err := terminal.AcquireTerminal()
	if err != nil {
		return err
	}
	defer term.Release()

	events := make(chan event)

	resizeCh := make(chan os.Signal, 1)
	signal.Notify(resizeCh, syscall.SIGWINCH)
	defer signal.Stop(resizeCh)

	go func() {
		for {
			input, err := term.ReadInput()
			if err != nil {
				if errors.Is(err, io.EOF) {
					events <- eventQuit{}
				} else {
					events <- eventQuit{err: err}
				}
				return
			}

			events <- eventInput{input: input}
		}
	}()

	srvConn := newServerConn(address)
	requests, err := srvConn.start(events)
	if err != nil {
		return err
	}

	logs := new(strings.Builder)
	ui := new(strings.Builder)

	m := newModel(requests)
	for {
		fmt.Print("\x1b7")
		m.render(logs, ui)
		fmt.Print(ui.String())
		ui.Reset()

		select {
		case <-resizeCh:
		case event := <-events:
			if quit, ok := event.(eventQuit); ok {
				return quit.err
			}

			var updateTask task
			m, updateTask = m.update(logs, event)

			if updateTask != nil {
				go func() {
					events <- updateTask()
				}()
			}
		}

		fmt.Print("\x1b8\x1b[0J")
		fmt.Print(logs.String())
		logs.Reset()
	}
}

type request struct {
	ack chan<- proto.ServerMsgAck
	msg proto.ClientMsgPayload
}

type serverConn struct {
	address          string
	inflightRequests map[uint32]request
	mutex            sync.Mutex
}

func newServerConn(address string) serverConn {
	return serverConn{
		address:          address,
		inflightRequests: make(map[uint32]request),
	}
}

func (s *serverConn) start(events chan<- event) (chan<- request, error) {
	conn, err := net.Dial("tcp", s.address)
	if err != nil {
		return nil, err
	}

	requests := make(chan request)

	go s.sendRequests(conn, requests, events)
	go s.receiveMsgs(conn, events)
	go s.keepAlive(requests)

	return requests, nil
}

func (s *serverConn) sendRequests(conn net.Conn, requests <-chan request, events chan<- event) {
	encoder := proto.NewClientEncoder(conn)
	for msg := range requests {
		err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			events <- eventQuit{err: err}
			break
		}

		id, err := encoder.Encode(msg.msg)
		if err != nil {
			events <- eventQuit{err: err}
			break
		}

		s.mutex.Lock()
		s.inflightRequests[id] = msg
		s.mutex.Unlock()
	}
}

func (s *serverConn) receiveMsgs(conn net.Conn, events chan<- event) {
	decoder := proto.NewServerDecoder(conn)
	for {
		msg, err := decoder.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) {
				events <- eventQuit{}
			} else {
				events <- eventQuit{err: err}
			}
			break
		}

		switch msg := msg.Payload.(type) {
		case *proto.ServerMsgAck:
			s.mutex.Lock()
			req := s.inflightRequests[msg.Id]
			delete(s.inflightRequests, msg.Id)
			s.mutex.Unlock()

			req.ack <- *msg
		case *proto.ServerMsgBroadcast:
			events <- eventBroadcast{msg: *msg}
		}
	}
}

func (s *serverConn) keepAlive(requests chan<- request) {
	for {
		time.Sleep(3 * time.Second)

		ack := make(chan proto.ServerMsgAck)

		requests <- request{
			ack: ack,
			msg: &proto.ClientMsgKeepAlive{},
		}

		<-ack
	}
}

type event any
type task func() event

type (
	eventQuit struct {
		err error
	}

	eventInput struct {
		input terminal.Input
	}

	eventUserInfoAck struct {
		ack  proto.ServerMsgAck
		name string
	}

	eventChatAck struct {
		ack proto.ServerMsgAck
	}

	eventBroadcast struct {
		msg proto.ServerMsgBroadcast
	}
)

func quit(err error) task {
	return func() event {
		return eventQuit{err: err}
	}
}

func sendUserInfo(requests chan<- request, msg proto.ClientMsgUserInfo) task {
	return func() event {
		ack := make(chan proto.ServerMsgAck)

		requests <- request{
			ack: ack,
			msg: &msg,
		}

		return eventUserInfoAck{ack: <-ack, name: msg.Name}
	}
}

func sendChat(requests chan<- request, msg proto.ClientMsgChat) task {
	return func() event {
		ack := make(chan proto.ServerMsgAck)

		requests <- request{
			ack: ack,
			msg: &msg,
		}

		return eventChatAck{ack: <-ack}
	}
}

type model struct {
	requests        chan<- request
	textEntry       textEntry
	username        string
	waitingResponse bool
}

func newModel(requests chan<- request) model {
	return model{
		requests:  requests,
		textEntry: newTextEntry(),
	}
}

func (m model) render(logs io.Writer, ui io.Writer) {
	if m.waitingResponse {
		fmt.Fprint(ui, "Waiting for server response...")
		return
	}

	if m.username == "" {
		fmt.Fprint(ui, "Your name: ")
	} else {
		fmt.Fprint(ui, "> ")
	}

	fmt.Fprint(ui, string(m.textEntry.Runes))
	fmt.Fprint(ui, strings.Repeat("\b", len(m.textEntry.Runes)-m.textEntry.Cursor))
}

func (m model) update(logs io.Writer, event event) (model, task) {
	switch event := event.(type) {
	case eventInput:
		if event.input.Kind == terminal.InputKindControl &&
			event.input.Str == "\x03" {
			return m, quit(nil)
		}

		if m.waitingResponse {
			return m, nil
		}

		switch event.input.Kind {
		case terminal.InputKindControl:
			switch event.input.Str {
			case "\r", "\n":
				input := string(m.textEntry.Runes)
				m.textEntry.clear()
				if m.username == "" {
					m.waitingResponse = true
					return m, sendUserInfo(m.requests, proto.ClientMsgUserInfo{Name: input})
				} else {
					m.waitingResponse = true
					return m, sendChat(m.requests, proto.ClientMsgChat{Text: input})
				}
			default:
				m.textEntry.handleInput(&event.input)
			}
		default:
			m.textEntry.handleInput(&event.input)
		}
	case eventBroadcast:
		fmt.Fprintf(
			logs,
			"%s (at %s): %s\r\n",
			event.msg.From,
			event.msg.TimeStamp.Format("15:04"),
			event.msg.Text,
		)
	case eventUserInfoAck:
		if event.ack.Ok {
			m.username = event.name
		} else {
			fmt.Fprintf(logs, "ERROR: %s\r\n", event.ack.Message)
		}

		m.waitingResponse = false
	case eventChatAck:
		if !event.ack.Ok {
			fmt.Fprintf(logs, "ERROR: %s\r\n", event.ack.Message)
		}

		m.waitingResponse = false
	}

	return m, nil
}
