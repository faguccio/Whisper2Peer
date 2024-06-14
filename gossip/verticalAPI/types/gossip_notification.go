package verticalapi

import (
	"encoding/binary"
	"errors"
	"slices"
)

// This type represents a GossipNotification packet in the verticalApi.
type GossipNotification struct {
	MessageHeader MessageHeader
	MessageId uint16
	DataType  GossipType
	Data      []byte
}

// // Unmarshals the GossipNotification packet from the provided buffer.
// //
// // Returns the number of bytes read from the buffer.
// func (e *GossipNotification) Unmarshal(buf []byte) (int, error) {
// 	return 0, ErrMethodNotImplemented
// }

// Marshals the GossipNotification packet to the provided buffer.
//
// If the provided buffer is too small, this function will just grow it.
func (e *GossipNotification) Marshal(buf []byte) ([]byte, error) {
	if e.MessageHeader.Type != GossipNotificationType {
		return nil, errors.New("wrong type")
	}

	buf = slices.Grow(buf, e.CalcSize())
	buf = buf[:e.CalcSize()]

	if err := e.MessageHeader.Marshal(buf); err != nil {
		return nil, err
	}

	idx := e.MessageHeader.CalcSize()

	binary.BigEndian.PutUint16(buf[idx:], e.MessageId)
	idx += 2

	binary.BigEndian.PutUint16(buf[idx:], uint16(e.DataType))
	idx += 2

	copy(buf[idx:], e.Data)
	idx += len(e.Data)

	return buf, nil
}

// Returns the size of the GossipNotification packet.
func (e *GossipNotification) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.MessageId)
	s += binary.Size(e.DataType)
	s += len(e.Data)
	return s
}
