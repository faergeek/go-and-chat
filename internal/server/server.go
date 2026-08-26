package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/faergeek/go-and-chat/internal/proto"
)

type client struct {
	id      string
	name    string
	receive <-chan proto.ClientMsg
	send    chan<- proto.ServerMsgPayload
}

type clientMessage struct {
	client *client
	msg    proto.ClientMsg
}

func Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("Server is listening on %s\n", listener.Addr())

	addClient := make(chan *client)
	go handleMessages(addClient)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting a connection: %s", err)
			continue
		}

		go handleConnection(conn, addClient)
	}
}

func handleMessages(addClient <-chan *client) {
	clientsByName := make(map[string]*client)
	deleteClient := make(chan *client)
	messages := make(chan clientMessage)

	for {
		select {
		case client := <-addClient:
			go func() {
				for msg := range client.receive {
					messages <- clientMessage{client: client, msg: msg}
				}

				deleteClient <- client
			}()
		case client := <-deleteClient:
			close(client.send)

			if client.name == "" {
				continue
			}

			delete(clientsByName, client.name)

			log.Printf("%s left the chat\n", client.name)

			broadcast := proto.ServerMsgBroadcast{
				From:      "System",
				Text:      fmt.Sprintf("%s left the chat", client.name),
				TimeStamp: time.Now(),
			}

			for _, otherClient := range clientsByName {
				otherClient.send <- &broadcast
			}
		case cm := <-messages:
			if _, ok := cm.msg.Payload.(*proto.ClientMsgKeepAlive); ok {
				log.Printf("%s: keep alive\n", cm.client.id)
				cm.client.send <- &proto.ServerMsgAck{
					Id: cm.msg.Id,
					Ok: true,
				}
				continue
			}

			if cm.client.name == "" {
				switch msg := cm.msg.Payload.(type) {
				case *proto.ClientMsgUserInfo:
					name := strings.Trim(msg.Name, " ")

					if name == "" {
						log.Printf("%s: sent empty name", cm.client.id)
						cm.client.send <- &proto.ServerMsgAck{
							Id:      cm.msg.Id,
							Message: "Empty names are not allowed",
						}
						continue
					}

					if !isPrintable(name) {
						log.Printf("%s: sent non-printable characters as a name: %v", cm.client.id, []byte(name))
						cm.client.send <- &proto.ServerMsgAck{
							Id:      cm.msg.Id,
							Message: "Non-printable characters in the name are not allowed",
						}
						continue
					}

					if _, taken := clientsByName[name]; taken {
						log.Printf("%s: chose a name that is already taken ('%s')", cm.client.id, name)
						cm.client.send <- &proto.ServerMsgAck{
							Id:      cm.msg.Id,
							Message: fmt.Sprintf("Name '%s' is already taken", name),
						}
						continue
					}

					log.Printf("%s: adding under a name '%s'", cm.client.id, name)
					cm.client.send <- &proto.ServerMsgAck{
						Id: cm.msg.Id,
						Ok: true,
					}

					cm.client.name = name
					clientsByName[name] = cm.client

					log.Printf("%s entered the chat\n", name)

					broadcast := proto.ServerMsgBroadcast{
						From:      "System",
						Text:      fmt.Sprintf("%s entered the chat", name),
						TimeStamp: time.Now(),
					}

					for _, otherClient := range clientsByName {
						if otherClient.name != name {
							otherClient.send <- &broadcast
						}
					}
				default:
					log.Printf("%s: sent an unexpected message: %#v", cm.client.id, msg)
					cm.client.send <- &proto.ServerMsgAck{
						Id:      cm.msg.Id,
						Message: "Nothing except keep alive and client info messages are allowed",
					}
				}
				continue
			}

			switch msg := cm.msg.Payload.(type) {
			case *proto.ClientMsgChat:
				text := strings.Trim(msg.Text, " ")

				if text == "" {
					log.Printf("%s: sent an empty message", cm.client.name)
					cm.client.send <- &proto.ServerMsgAck{
						Id:      cm.msg.Id,
						Message: "Empty messages are not allowed",
					}
					continue
				}

				if !isPrintable(text) {
					log.Printf("%s: sent a message with non-printable characters: %v", cm.client.name, []byte(text))
					cm.client.send <- &proto.ServerMsgAck{
						Id:      cm.msg.Id,
						Message: "Non-printable characters in message text are not allowed",
					}
					continue
				}

				log.Printf("%s: sent a message: '%s'", cm.client.name, text)
				cm.client.send <- &proto.ServerMsgAck{
					Id: cm.msg.Id,
					Ok: true,
				}

				broadcast := proto.ServerMsgBroadcast{
					From:      cm.client.name,
					Text:      text,
					TimeStamp: time.Now(),
				}

				for _, client := range clientsByName {
					client.send <- &broadcast
				}
			default:
				log.Printf("%s: sent an unexpected message: %#v", cm.client.name, msg)
				cm.client.send <- &proto.ServerMsgAck{
					Id:      cm.msg.Id,
					Message: "Nothing except keep alive and chat messages are allowed",
				}
			}
		}
	}
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}

	return true
}

func handleConnection(conn net.Conn, addClient chan<- *client) {
	defer conn.Close()

	id := conn.RemoteAddr().String()

	log.Printf("%s: connected\n", id)
	defer log.Printf("%s: disconnected\n", id)

	receive := make(chan proto.ClientMsg)
	receiveDone := make(chan error)
	go receiveMessages(conn, receive, receiveDone)

	send := make(chan proto.ServerMsgPayload)
	sendDone := make(chan error)
	go sendMessages(conn, send, sendDone)

	addClient <- &client{
		id:      id,
		receive: receive,
		send:    send,
	}

	if err := <-receiveDone; err != nil {
		log.Printf("%s: %s\n", id, err)
	}

	if err := <-sendDone; err != nil {
		log.Printf("%s: %s\n", id, err)
	}
}

func receiveMessages(conn net.Conn, out chan<- proto.ClientMsg, done chan<- error) {
	defer close(out)

	decoder := proto.NewClientDecoder(conn)
	for {
		err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			done <- fmt.Errorf("Error setting a read deadline: %w", err)
			return
		}

		msg, err := decoder.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) {
				done <- nil
				return
			}

			if errors.Is(err, os.ErrDeadlineExceeded) {
				done <- fmt.Errorf("Read timed out")
				return
			}

			done <- fmt.Errorf("Error receiving a message from client: %w", err)
			return
		}

		out <- msg
	}
}

func sendMessages(conn net.Conn, in <-chan proto.ServerMsgPayload, done chan<- error) {
	encoder := proto.NewServerEncoder(conn)
	for msg := range in {
		err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			done <- fmt.Errorf("Error setting a write deadline: %w", err)
			return
		}

		err = encoder.Encode(msg)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				done <- fmt.Errorf("Write timed out")
				return
			}

			done <- fmt.Errorf("Error sending a message to client: %w", err)
			return
		}
	}

	done <- nil
}
