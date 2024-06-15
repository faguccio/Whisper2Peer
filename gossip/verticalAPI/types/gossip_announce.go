package verticalapi

import (
	"encoding/binary"
	"errors"
	"gossip/common"
)

// This type represents a GossipAnnounce packet in the verticalApi.
type GossipAnnounce struct {
	Ga            common.GossipAnnounce
	MessageHeader MessageHeader
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

	e.Ga.TTL = buf[idx]
	idx += 1

	e.Ga.Reserved = buf[idx]
	idx += 1

	e.Ga.DataType = common.GossipType(binary.BigEndian.Uint16(buf[idx:]))
	idx += 2

	// golang slices: [a:b] index b is excluded
	// FABIO: do we accept smaller data frame than expected???
	e.Ga.Data = buf[idx:min(int(e.MessageHeader.Size), len(buf))]
	idx += len(e.Ga.Data)

	return idx, nil
}

// Marshals the GossipAnnounce packet to the provided buffer.
// func (e *GossipAnnounce) Marshal(buf []byte) error {
// 	return ErrMethodNotImplemented
// }

// Returns the size of the GossipAnnounce packet.
func (e *GossipAnnounce) CalcSize() int {
	return e.MessageHeader.CalcSize() + e.Ga.CalcSize()
}

// Mark this type as vertical type
func (e *GossipAnnounce) isVertType() {}
