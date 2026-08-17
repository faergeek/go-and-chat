package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type srvClient struct {
	conn     net.Conn
	encoder  *gob.Encoder
	decoder  *gob.Decoder
	username string
}

type server struct {
	clients map[string]*srvClient
	mutex   sync.RWMutex
}

func runServer(address string) error {
	fmt.Printf("Attempting to start a server on %s...", address)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("FAILED")
		return err
	}
	fmt.Println("SUCCESS")
	defer listener.Close()

	fmt.Printf("Server is listening on %s\n", listener.Addr())

	srv := server{clients: make(map[string]*srvClient)}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept a connection: %v", err.Error())
			continue
		}

		go srv.handleClientConnection(conn)
	}
}

func (s *server) handleClientConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		fmt.Printf("Closing a connection from %s\n", conn.RemoteAddr())
	}()

	fmt.Printf("Got a connection from %s\n", conn.RemoteAddr())

	c := srvClient{
		conn:    conn,
		encoder: gob.NewEncoder(conn),
		decoder: gob.NewDecoder(conn),
	}

	defer func() {
		s.mutex.Lock()
		fmt.Printf("Deleting '%s' client from map\n", c.username)
		delete(s.clients, c.username)
		s.mutex.Unlock()
	}()

	for {
		msg, err := c.receiveMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error receiving a message from client: %v\n", err.Error())
			return
		}

		err = s.handleClientMessage(&c, msg)
		if err != nil {
			return
		}
	}
}

func (c *srvClient) receiveMessage() (Msg, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("Could not set a connection read deadline: %v\n", err)
	}

	msg, err := DecodeMsg(c.decoder)
	if err != nil {
		return nil, fmt.Errorf("Error receiving a message from client: %w\n", err)
	}

	return msg, nil
}

func (c *srvClient) sendMessage(msg Msg) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	if err := EncodeMsg(c.encoder, msg); err != nil {
		return err
	}

	return nil
}

func (c *srvClient) sendAck() error {
	return c.sendMessage(&MsgAck{Ok: true})
}

func (c *srvClient) sendNack(message string) error {
	return c.sendMessage(&MsgAck{Ok: false, Message: message})
}

func (s *server) handleClientMessage(c *srvClient, msg Msg) error {
	switch m := msg.(type) {
	case *MsgPing:
		return c.sendAck()
	case *MsgClientInfo:
		fmt.Printf("Got client info: %#v\n", m)

		if m.Username == "" {
			return c.sendNack("Empty names are not allowed")
		}

		s.mutex.RLock()
		_, taken := s.clients[m.Username]
		s.mutex.RUnlock()

		if taken {
			return c.sendNack("Name is already taken")
		}

		c.username = m.Username

		s.mutex.Lock()
		s.clients[m.Username] = c
		s.mutex.Unlock()

		return c.sendAck()
	case *MsgChatMessage:
		if c.username == "" {
			return c.sendNack("Username must be provided first")
		}

		err := c.sendAck()
		if err != nil {
			return err
		}

		msg := *m
		msg.Username = c.username

		s.mutex.RLock()
		for _, client := range s.clients {
			err := client.sendMessage(&msg)
			if err != nil {
				return err
			}
		}
		s.mutex.RUnlock()

		return nil
	default:
		fmt.Printf("Got message from client: %#v\n", m)
		return c.sendNack("Unexpected message kind")
	}
}
