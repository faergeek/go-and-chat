package main

import (
	"encoding/gob"
	"fmt"
	"time"
)

type MsgKind int

const (
	MsgKindPing MsgKind = iota
	MsgKindAck
	MsgKindClientInfo
	MsgKindChatMessage
)

type Msg interface {
	Kind() MsgKind
}

type MsgPing struct{}

func (m *MsgPing) Kind() MsgKind {
	return MsgKindPing
}

type MsgAck struct {
	Ok      bool
	Message string
}

func (m *MsgAck) Kind() MsgKind {
	return MsgKindAck
}

type MsgClientInfo struct {
	Username string
}

func (m *MsgClientInfo) Kind() MsgKind {
	return MsgKindClientInfo
}

type MsgChatMessage struct {
	AckAt    time.Time
	Message  string
	Username string
}

func (m *MsgChatMessage) Kind() MsgKind {
	return MsgKindChatMessage
}

func EncodeMsg(encoder *gob.Encoder, msg Msg) error {
	kind := msg.Kind()
	if err := encoder.Encode(kind); err != nil {
		return err
	}

	return encoder.Encode(msg)
}

func DecodeMsg(decoder *gob.Decoder) (Msg, error) {
	var kind MsgKind
	if err := decoder.Decode(&kind); err != nil {
		return nil, err
	}

	switch kind {
	case MsgKindPing:
		var msg MsgPing
		err := decoder.Decode(&msg)
		return &msg, err
	case MsgKindAck:
		var msg MsgAck
		err := decoder.Decode(&msg)
		return &msg, err
	case MsgKindClientInfo:
		var msg MsgClientInfo
		err := decoder.Decode(&msg)
		return &msg, err
	case MsgKindChatMessage:
		var msg MsgChatMessage
		err := decoder.Decode(&msg)
		return &msg, err
	default:
		return nil, fmt.Errorf("Unexpected message kind: %v", kind)
	}
}
