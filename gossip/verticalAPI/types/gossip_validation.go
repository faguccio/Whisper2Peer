package verticalapi

import (
	"encoding/binary"
	"errors"
	"gossip/common"
)

// This type represents a GossipValidation packet in the verticalApi.
type GossipValidation struct {
	Gv            common.GossipValidation
	MessageHeader MessageHeader
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

	e.Gv.MessageId = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.Gv.Bitfield = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	// just for conveniance
	e.Gv.Valid = e.Gv.Bitfield&0x1 == 0x1

	return idx, nil
}

// // Marshals the GossipValidation packet to the provided buffer.
// func (e *GossipValidation) Marshal(buf []byte) error {
// 	return ErrMethodNotImplemented
// }

// Returns the size of the GossipValidation packet.
func (e *GossipValidation) CalcSize() int {
	return e.MessageHeader.CalcSize() + e.Gv.CalcSize()
}

// Mark this type as vertical type
func (e *GossipValidation) isVertType() {}
