package verticalapi

import (
	"encoding/binary"
	"errors"
)

const GossipValidationType = 503

type GossipValidation struct {
	MessageHeader
	MessageId uint16
	Bitfield  uint16
	valid     bool
}

func (e *GossipValidation) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != GossipValidationType {
		return 0, errors.New("wrong type")
	}
	idx := e.MessageHeader.CalcSize()

	e.MessageId = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.Bitfield = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.valid = e.Bitfield&0x1 == 0x1

	return 2 + 2, nil
}
func (e *GossipValidation) CalcSize() int {
	return binary.Size(e)
}
