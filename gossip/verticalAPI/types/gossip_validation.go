package verticalapi

import (
	"encoding/binary"
	"errors"
)

// This type represents a GossipValidation packet in the verticalApi.
type GossipValidation struct {
	MessageHeader MessageHeader
	MessageId     uint16
	Bitfield      uint16
	// only for ease of use we extract this from the bitfield on Unmarshal
	Valid bool
}

// Unmarshals the GossipValidation packet from the provided buffer.
//
// Returns the number of bytes read from the buffer.
func (e *GossipValidation) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != GossipValidationType {
		return 0, errors.New("wrong type")
	}

	idx := e.MessageHeader.CalcSize()

	//Either I did not understand or the function was faulty
	if len(buf) < e.CalcSize() {
		return 0, ErrNotEnoughData
	}

	e.MessageId = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.Bitfield = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	// just for conveniance
	e.Valid = e.Bitfield&0x1 == 0x1

	return idx, nil
}

// // Marshals the GossipValidation packet to the provided buffer.
// func (e *GossipValidation) Marshal(buf []byte) error {
// 	return ErrMethodNotImplemented
// }

// Returns the size of the GossipValidation packet.
func (e *GossipValidation) CalcSize() int {
	return binary.Size(e) - binary.Size(e.Valid)
}

func (e *GossipValidation) SetValid(v bool) {
	e.Valid = v
	if v {
		e.Bitfield = e.Bitfield | (uint16(1) << 0)
	} else {
		e.Bitfield = e.Bitfield & ^(uint16(1) << 0)
	}
}
