package verticalapi

import (
	"encoding/binary"
	"errors"
)

const GossipAnnounceType = 500

type GossipAnnounce struct {
	MessageHeader
	TTL      uint8
	Reserved uint8
	DataType uint16
	Data     []byte
}

func (e *GossipAnnounce) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != GossipAnnounceType {
		return 0, errors.New("wrong type")
	}

	idx := e.MessageHeader.CalcSize()

	e.TTL = buf[idx]
	idx += 1

	e.Reserved = buf[idx]
	idx += 1

	e.DataType = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	// golang slices: [a:b] index b is excluded
	e.Data = buf[idx:min(int(e.MessageHeader.Size), len(buf))]

	return 1 + 1 + 2 + len(e.Data), nil
}
func (e *GossipAnnounce) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.TTL)
	s += binary.Size(e.Reserved)
	s += binary.Size(e.DataType)
	s += len(e.Data)
	return s
}
