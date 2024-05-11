package verticalapi

import (
	"encoding/binary"
	"errors"
	"slices"
)

const GossipNotificationType = 502

type GossipNotification struct {
	MessageHeader
	MessageId uint16
	DataType  uint16
	Data      []byte
}

func (e *GossipNotification) Marshal(buf []byte) ([]byte, error) {
	if e.MessageHeader.Type != GossipNotificationType {
		return nil, errors.New("wrong type")
	}

	buf = slices.Grow(buf, e.CalcSize()+3*2)
	buf = buf[:e.CalcSize()+3*2]
	if err := e.MessageHeader.Marshal(buf); err != nil {
		return nil, err
	}

	idx := e.MessageHeader.CalcSize()

	binary.BigEndian.PutUint16(buf[idx:], e.MessageId)
	idx += 2

	binary.BigEndian.PutUint16(buf[idx:], e.DataType)
	idx += 2

	copy(buf[idx:], e.Data)
	idx += len(e.Data)

	return buf, nil
}
func (e *GossipNotification) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.MessageId)
	s += binary.Size(e.DataType)
	s += len(e.Data)
	return s
}
