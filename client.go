package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/faergeek/go-and-chat/internal/terminal"
)

type client struct {
	conn       net.Conn
	fromServer chan Msg
	toServer   chan Msg
	errCh      chan error
}

func runClient(address string) error {
	if !terminal.IsTerminal() {
		fmt.Fprintln(os.Stderr, "Non-interactive use is not supported")
		os.Exit(1)
	}

	term, err := terminal.AcquireTerminal()
	if err != nil {
		panic(err)
	}
	defer term.Release()

	resizeCh := make(chan os.Signal, 1)
	signal.Notify(resizeCh, syscall.SIGWINCH)
	defer signal.Stop(resizeCh)

	termInputCh := make(chan terminal.Input)
	termInputErrCh := make(chan error)

	go func() {
		for {
			input, err := term.ReadInput()
			if err != nil {
				termInputErrCh <- err
				close(termInputCh)
				return
			}

			termInputCh <- input
		}
	}()

	fmt.Printf("Attempting to connect to a server on %s...", address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Print("FAILED\r\n")
		return err
	}
	fmt.Print("SUCCESS\r\n")
	defer conn.Close()

	fmt.Printf("Connected to a server at %s\r\n", conn.RemoteAddr())

	client := NewClient(conn)
	go client.startReader()
	go client.startWriter()

	var username string
	textInput := newTextInput()
	for {
		if username == "" {
			fmt.Print("\x1b7")
			fmt.Print("Your name: ")
			fmt.Print(string(textInput.Runes))
			fmt.Print(strings.Repeat("\b", len(textInput.Runes)-textInput.Cursor))
		} else {
			fmt.Print("\x1b7")
			fmt.Printf("%s > ", username)
			fmt.Print(string(textInput.Runes))
			fmt.Print(strings.Repeat("\b", len(textInput.Runes)-textInput.Cursor))
		}

		select {
		case <-resizeCh:
			fmt.Print("\x1b8\x1b[0J")
		case termInput := <-termInputCh:
			fmt.Print("\x1b8\x1b[0J")

			done, err := textInput.handleInput(&termInput)
			if err != nil {
				if err == io.EOF {
					return nil
				}

				return err
			}

			if done {
				input := string(textInput.Runes)

				if username == "" {
					ack, err := client.sendMessage(&MsgClientInfo{Username: input})
					if err != nil {
						return err
					}

					if ack.Ok {
						username = input
						textInput.clear()
					} else {
						fmt.Printf("ERROR: %s\r\n", ack.Message)
					}
				} else {
					ack, err := client.sendMessage(&MsgChatMessage{Message: string(textInput.Runes)})
					if err != nil {
						return err
					}

					if ack.Ok {
						textInput.clear()
					} else {
						fmt.Printf("%s\r\n", ack.Message)
					}
				}
			}
		case err := <-termInputErrCh:
			fmt.Print("\x1b8\x1b[0J")

			return fmt.Errorf("Could not read input: %w", err)
		case srvMsg := <-client.fromServer:
			fmt.Print("\x1b8\x1b[0J")

			chatMsg, ok := srvMsg.(*MsgChatMessage)
			if !ok {
				return fmt.Errorf("Unexpected to only receive chat messages, got %#v", srvMsg)
			}

			fmt.Printf(
				"\r%s (%v): %s\r\n",
				chatMsg.Username,
				chatMsg.AckAt.Format("15:04"),
				chatMsg.Message,
			)
		case srvErr := <-client.errCh:
			fmt.Print("\x1b8\x1b[0J")

			if srvErr != io.EOF {
				fmt.Printf("Got error from server: %v\r\n", srvErr)
			}

			return srvErr
		case <-time.After(2 * time.Second):
			fmt.Print("\x1b8\x1b[0J")

			_, err := client.sendMessage(&MsgPing{})
			if err != nil {
				return err
			}
		}
	}
}

func NewClient(conn net.Conn) client {
	readCh := make(chan Msg)
	writeCh := make(chan Msg)
	errCh := make(chan error)

	return client{
		conn:       conn,
		fromServer: readCh,
		toServer:   writeCh,
		errCh:      errCh,
	}
}

func (c *client) startReader() {
	defer close(c.fromServer)

	decoder := gob.NewDecoder(c.conn)

	for {
		msg, err := DecodeMsg(decoder)
		if err != nil {
			c.errCh <- err
			return
		}

		c.fromServer <- msg
	}
}

func (c *client) startWriter() {
	encoder := gob.NewEncoder(c.conn)

	for msg := range c.toServer {
		if err := EncodeMsg(encoder, msg); err != nil {
			c.errCh <- err
			return
		}
	}
}

func (c *client) sendMessage(msg Msg) (*MsgAck, error) {
	c.toServer <- msg

	select {
	case response := <-c.fromServer:
		ack, ok := response.(*MsgAck)
		if !ok {
			return nil, fmt.Errorf("Expected to receive an ack, got: %#v", response)
		}

		return ack, nil
	case err := <-c.errCh:
		return nil, err
	}
}
