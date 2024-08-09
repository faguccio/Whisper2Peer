package verticalapi

import (
	"encoding/binary"
	"errors"
	"gossip/common"
	"slices"
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
func (e *GossipAnnounce) Marshal(buf []byte) ([]byte, error) {
	if e.MessageHeader.Type != GossipAnnounceType {
		return nil, errors.New("wrong type")
	}

	buf = slices.Grow(buf, e.CalcSize())
	buf = buf[:e.CalcSize()]

	if err := e.MessageHeader.Marshal(buf); err != nil {
		return nil, err
	}

	idx := e.MessageHeader.CalcSize()

	buf[idx] = e.Ga.TTL
	idx += 1
	// skip reserved field
	idx += 1

	binary.BigEndian.PutUint16(buf[idx:], uint16(e.Ga.DataType))
	idx += 2

	copy(buf[idx:], e.Ga.Data)
	idx += len(e.Ga.Data)

	return buf, nil
}

// Returns the size of the GossipAnnounce packet.
func (e *GossipAnnounce) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.Ga.TTL)
	s += binary.Size(e.Ga.Reserved)
	s += binary.Size(e.Ga.DataType)
	s += len(e.Ga.Data)
	return s
}

// Mark this type as vertical type
func (e *GossipAnnounce) isVertType() {}
