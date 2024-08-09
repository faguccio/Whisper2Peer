package verticalapi

import (
	"encoding/binary"
	"errors"
	"gossip/common"
	"slices"
)

// This type represents a GossipNotify packet in the verticalApi.
type GossipNotify struct {
	Gn            common.GossipNotify
	MessageHeader MessageHeader
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

	e.Gn.Reserved = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.Gn.DataType = common.GossipType(binary.BigEndian.Uint16(buf[idx:]))
	idx += 2

	return idx, nil
}

// Marshals the GossipNotify packet to the provided buffer.
func (e *GossipNotify) Marshal(buf []byte) ([]byte, error) {
	if e.MessageHeader.Type != GossipNotifyType {
		return nil, errors.New("wrong type")
	}

	buf = slices.Grow(buf, e.CalcSize())
	buf = buf[:e.CalcSize()]

	if err := e.MessageHeader.Marshal(buf); err != nil {
		return nil, err
	}

	idx := e.MessageHeader.CalcSize()

	// skip reserved
	idx += 2

	binary.BigEndian.PutUint16(buf[idx:], uint16(e.Gn.DataType))
	idx += 2

	return buf, nil
}

// Returns the size of the GossipNotify packet.
func (e *GossipNotify) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.Gn.DataType)
	s += binary.Size(e.Gn.Reserved)
	return s
}

// Mark this type as vertical type
func (e *GossipNotify) isVertType() {}
