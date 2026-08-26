package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

type ClientMsgKind byte

const (
	ClientMsgKindKeepAlive ClientMsgKind = iota
	ClientMsgKindUserInfo
	ClientMsgKindChat
)

type clientMsgHeader struct {
	Id     uint32
	Kind   ClientMsgKind
	Length uint8
}

const maxPayloadSize = 256

type ClientMsgPayload interface {
	encode() ([]byte, error)
	decode(data []byte) error
}

type ClientMsg struct {
	Id      uint32
	Payload ClientMsgPayload
}

type ClientMsgKeepAlive struct{}

func (m *ClientMsgKeepAlive) encode() ([]byte, error) {
	return []byte{}, nil
}

func (m *ClientMsgKeepAlive) decode(data []byte) error {
	return nil
}

type ClientMsgUserInfo struct{ Name string }

func (m *ClientMsgUserInfo) encode() ([]byte, error) {
	data := make([]byte, 0)
	data, err := appendString(data, m.Name)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (m *ClientMsgUserInfo) decode(data []byte) error {
	var err error
	data, m.Name, err = readString(data)
	return err
}

type ClientMsgChat struct{ Text string }

func (m *ClientMsgChat) encode() ([]byte, error) {
	data := make([]byte, 0)
	data, err := appendString(data, m.Text)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (m *ClientMsgChat) decode(data []byte) error {
	var err error
	data, m.Text, err = readString(data)
	return err
}

type ClientEncoder struct {
	nextId uint32
	writer io.Writer
}

func NewClientEncoder(writer io.Writer) ClientEncoder {
	return ClientEncoder{
		nextId: 1,
		writer: writer,
	}
}

func (c *ClientEncoder) Encode(msg ClientMsgPayload) (uint32, error) {
	id := c.nextId
	c.nextId++

	var kind ClientMsgKind
	switch msg := msg.(type) {
	case *ClientMsgKeepAlive:
		kind = ClientMsgKindKeepAlive
	case *ClientMsgUserInfo:
		kind = ClientMsgKindUserInfo
	case *ClientMsgChat:
		kind = ClientMsgKindChat
	default:
		return id, fmt.Errorf("Unexpected message: %#v", msg)
	}

	data, err := msg.encode()
	if err != nil {
		return id, err
	}

	length := len(data)
	if length > maxPayloadSize {
		return id, fmt.Errorf("Body is too large. Got: %d. Max: %d", length, maxPayloadSize)
	}

	header := clientMsgHeader{
		Id:     id,
		Kind:   kind,
		Length: uint8(length),
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, &header)
	buf.Write(data)

	_, err = c.writer.Write(buf.Bytes())
	if err != nil {
		return id, fmt.Errorf("Failed to write a message: %w", err)
	}

	return id, nil
}

type ClientDecoder struct {
	reader io.Reader
	mutex  sync.Mutex
}

func NewClientDecoder(reader io.Reader) ClientDecoder {
	return ClientDecoder{
		reader: reader,
	}
}

func (c *ClientDecoder) Decode() (ClientMsg, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var header clientMsgHeader
	err := binary.Read(c.reader, binary.BigEndian, &header)
	if err != nil {
		return ClientMsg{}, err
	}

	var payload ClientMsgPayload
	switch header.Kind {
	case ClientMsgKindKeepAlive:
		payload = &ClientMsgKeepAlive{}
	case ClientMsgKindUserInfo:
		payload = &ClientMsgUserInfo{}
	case ClientMsgKindChat:
		payload = &ClientMsgChat{}
	default:
		return ClientMsg{}, fmt.Errorf("Unexpected message kind: %v", header.Kind)
	}

	data := make([]byte, header.Length)
	_, err = io.ReadFull(c.reader, data)
	if err != nil {
		return ClientMsg{}, err
	}

	err = payload.decode(data)
	if err != nil {
		return ClientMsg{}, err
	}

	msg := ClientMsg{
		Id:      header.Id,
		Payload: payload,
	}

	return msg, nil
}
