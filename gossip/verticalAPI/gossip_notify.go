package verticalapi

import (
	"encoding/binary"
	"errors"
)

const GossipNotifyType = 501

type GossipNotify struct {
	MessageHeader
	Reserved uint16
	DataType uint16
}

func (e *GossipNotify) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != GossipNotifyType {
		return 0, errors.New("wrong type")
	}

	idx := e.MessageHeader.CalcSize()

	e.Reserved = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.DataType = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	return 2 + 2, nil
}
func (e *GossipNotify) CalcSize() int {
	return binary.Size(e)
}
