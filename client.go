package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type client struct {
	conn       net.Conn
	fromServer chan Msg
	toServer   chan Msg
	errCh      chan error
}

func runClient(address string) error {
	fmt.Printf("Attempting to connect to a server on %s...", address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("FAILED")
		return err
	}
	defer conn.Close()

	fmt.Println("SUCCESS")

	fmt.Printf("Connected to a server at %s\n", conn.RemoteAddr())

	stdinCh := make(chan string)
	stdinErrCh := make(chan error)
	go readStdin(stdinCh, stdinErrCh)

	client := NewClient(conn)
	go client.startReader()
	go client.startWriter()

	usernamePrompt := "Your username: "
	chatPrompt := "> "
	fmt.Print(usernamePrompt)

	var username string
	for {
		select {
		case input := <-stdinCh:
			if username == "" {
				ack, err := client.sendMessage(&MsgClientInfo{Username: input})
				if err != nil {
					return err
				}

				if ack.Ok {
					username = input
					fmt.Print(chatPrompt)
				} else {
					fmt.Println(ack.Message)
					fmt.Print(usernamePrompt)
				}
			} else {
				ack, err := client.sendMessage(&MsgChatMessage{Message: input})
				if err != nil {
					return err
				}

				if !ack.Ok {
					fmt.Println(ack.Message)
				}

				fmt.Print(chatPrompt)
			}
		case err := <-stdinErrCh:
			fmt.Printf("Could not read from stdin: %v", err)
			return err
		case srvMsg := <-client.fromServer:
			chatMsg, ok := srvMsg.(*MsgChatMessage)
			if !ok {
				return fmt.Errorf("Unexpected to only receive chat messages, got %#v\n", srvMsg)
			}

			fmt.Println(strings.Repeat("\r", len(chatPrompt)))
			fmt.Printf("%s: %s\n", chatMsg.Username, chatMsg.Message)
			fmt.Print(chatPrompt)
		case srvErr := <-client.errCh:
			if srvErr != io.EOF {
				fmt.Printf("Got error from server: %v\n", srvErr)
			}

			return srvErr
		case <-time.After(2 * time.Second):
			_, err := client.sendMessage(&MsgPing{})
			if err != nil {
				return err
			}
		}
	}
}

func readStdin(ch chan<- string, errch chan<- error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			errch <- err
			return
		}

		ch <- input[:len(input)-1]
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
