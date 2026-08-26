package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"
)

type ServerMsgKind byte

const (
	ServerMsgKindAck = iota
	ServerMsgKindBroadcast
)

type serverMsgHeader struct {
	Kind   ServerMsgKind
	Length uint16
}

type ServerMsgPayload interface {
	encode() ([]byte, error)
	decode(data []byte) error
}

type ServerMsg struct {
	Payload ServerMsgPayload
}

type ServerMsgAck struct {
	Id      uint32
	Ok      bool
	Message string
}

func (m *ServerMsgAck) encode() ([]byte, error) {
	data := make([]byte, 0)
	data = binary.BigEndian.AppendUint32(data, m.Id)

	if m.Ok {
		data = append(data, 1)
	} else {
		data = append(data, 0)
	}

	data, err := appendString(data, m.Message)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (m *ServerMsgAck) decode(data []byte) error {
	if length := len(data); length < 4 {
		return fmt.Errorf("unable to decode ack id. Expected %d bytes, got only %d", 4, length)
	}

	m.Id = binary.BigEndian.Uint32(data[:4])
	data = data[4:]

	if length := len(data); length < 1 {
		return fmt.Errorf("unable to decode ack ok flag. Not enough bytes")
	}

	ok := data[0]
	data = data[1:]
	m.Ok = ok != 0

	var err error
	data, m.Message, err = readString(data)
	if err != nil {
		return err
	}

	return nil
}

type ServerMsgBroadcast struct {
	From      string
	Text      string
	TimeStamp time.Time
}

func (m *ServerMsgBroadcast) encode() ([]byte, error) {
	data := make([]byte, 0)

	data, err := appendString(data, m.From)
	if err != nil {
		return nil, err
	}

	data, err = appendString(data, m.Text)
	if err != nil {
		return nil, err
	}

	timeBytes, err := m.TimeStamp.MarshalBinary()
	if err != nil {
		return nil, err
	}

	data = append(data, timeBytes...)

	return data, nil
}

func (m *ServerMsgBroadcast) decode(data []byte) error {
	var err error
	data, m.From, err = readString(data)
	if err != nil {
		return err
	}

	data, m.Text, err = readString(data)
	if err != nil {
		return err
	}

	err = m.TimeStamp.UnmarshalBinary(data)
	if err != nil {
		return err
	}

	return nil
}

type ServerEncoder struct {
	writer io.Writer
}

func NewServerEncoder(writer io.Writer) ServerEncoder {
	return ServerEncoder{
		writer: writer,
	}
}

func (s *ServerEncoder) Encode(msg ServerMsgPayload) error {
	var kind ServerMsgKind
	switch msg := msg.(type) {
	case *ServerMsgAck:
		kind = ServerMsgKindAck
	case *ServerMsgBroadcast:
		kind = ServerMsgKindBroadcast
	default:
		return fmt.Errorf("Unexpected message: %#v", msg)
	}

	data, err := msg.encode()
	if err != nil {
		return err
	}

	header := serverMsgHeader{
		Kind:   kind,
		Length: uint16(len(data)),
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, &header)
	buf.Write(data)

	_, err = s.writer.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("Failed to write a message: %w", err)
	}

	return nil
}

type ServerDecoder struct {
	mutex  sync.Mutex
	reader io.Reader
}

func NewServerDecoder(reader io.Reader) ServerDecoder {
	return ServerDecoder{
		reader: reader,
	}
}

func (s *ServerDecoder) Decode() (ServerMsg, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var header serverMsgHeader
	err := binary.Read(s.reader, binary.BigEndian, &header)
	if err != nil {
		return ServerMsg{}, err
	}

	var payload ServerMsgPayload
	switch header.Kind {
	case ServerMsgKindAck:
		payload = &ServerMsgAck{}
	case ServerMsgKindBroadcast:
		payload = &ServerMsgBroadcast{}
	default:
		return ServerMsg{}, fmt.Errorf("Unexpected message kind: %v", header.Kind)
	}

	data := make([]byte, header.Length)
	_, err = io.ReadFull(s.reader, data)
	if err != nil {
		return ServerMsg{}, err
	}

	err = payload.decode(data)
	if err != nil {
		return ServerMsg{}, err
	}

	msg := ServerMsg{
		Payload: payload,
	}

	return msg, nil
}
