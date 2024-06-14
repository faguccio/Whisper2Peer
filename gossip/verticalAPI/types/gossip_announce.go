package verticalapi

import (
	"encoding/binary"
	"errors"
)

// This type represents a GossipAnnounce packet in the verticalApi.
type GossipAnnounce struct {
	MessageHeader MessageHeader
	TTL      uint8
	Reserved uint8
	DataType GossipType
	Data     []byte
}

// Unmarshals the GossipAnnounce packet from the provided buffer.
//
// Returns the number of bytes read from the buffer.
func (e *GossipAnnounce) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != GossipAnnounceType {
		return 0, errors.New("wrong type")
	}

	if len(buf) < e.CalcSize() {
		return 0, ErrNotEnoughData
	}

	idx := e.MessageHeader.CalcSize()

	e.TTL = buf[idx]
	idx += 1

	e.Reserved = buf[idx]
	idx += 1

	e.DataType = GossipType(binary.BigEndian.Uint16(buf[idx:]))
	idx += 2

	// golang slices: [a:b] index b is excluded
	// FABIO: do we accept smaller data frame than expected???
	e.Data = buf[idx:min(int(e.MessageHeader.Size), len(buf))]
	idx += len(e.Data)

	return idx, nil
}

// Marshals the GossipAnnounce packet to the provided buffer.
// func (e *GossipAnnounce) Marshal(buf []byte) error {
// 	return ErrMethodNotImplemented
// }

// Returns the size of the GossipAnnounce packet.
func (e *GossipAnnounce) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.TTL)
	s += binary.Size(e.Reserved)
	s += binary.Size(e.DataType)
	s += len(e.Data)
	return s
}
