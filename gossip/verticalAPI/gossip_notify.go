package verticalapi

import (
	"encoding/binary"
	"errors"
)

// This type represents a GossipNotify packet in the verticalApi.
type GossipNotify struct {
	MessageHeader
	Reserved uint16
	DataType GossipType
}

// Unmarshals the GossipNotify packet from the provided buffer.
//
// Returns the number of bytes read from the buffer.
func (e *GossipNotify) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != GossipNotifyType {
		return 0, errors.New("wrong type")
	}

	if len(buf) < e.CalcSize() {
		return 0, ErrNotEnoughData
	}

	idx := e.MessageHeader.CalcSize()

	e.Reserved = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.DataType = GossipType(binary.BigEndian.Uint16(buf[idx:]))
	idx += 2

	return idx, nil
}

// Marshals the GossipNotify packet to the provided buffer.
//
// Not implemented for this message type, but needed to shadow the method from [MessageHeader].
func (e *GossipNotify) Marshal(buf []byte) error {
	return ErrMethodNotImplemented
}

// Returns the size of the GossipNotify packet.
func (e *GossipNotify) CalcSize() int {
	return binary.Size(e)
}
