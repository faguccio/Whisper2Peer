package verticalapi

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// This type represents a GossipValidation packet in the verticalApi.
type GossipValidation struct {
	MessageHeader
	MessageId uint16
	Bitfield  uint16
	// only for ease of use we extract this from the bitfield on Unmarshal
	valid bool
}

// Unmarshals the GossipValidation packet from the provided buffer.
//
// Returns the number of bytes read from the buffer.
func (e *GossipValidation) Unmarshal(buf []byte) (int, error) {
	e.MessageHeader.Unmarshal(buf)

	if e.MessageHeader.Type != GossipValidationType {
		return 0, errors.New("wrong type")
	}

	idx := e.MessageHeader.CalcSize()

	//Either I did not understand or the function was faulty
	fmt.Println(e.CalcSize()-e.MessageHeader.CalcSize(), len(buf[idx:]))
	if len(buf[idx:]) < e.CalcSize()-e.MessageHeader.CalcSize() {
		return 0, ErrNotEnoughData
	}

	e.MessageId = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.Bitfield = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	// just for conveniance
	e.valid = e.Bitfield&0x1 == 0x1

	return idx - e.MessageHeader.CalcSize(), nil
}

// Marshals the GossipValidation packet to the provided buffer.
//
// Not implemented for this message type, but needed to shadow the method from [MessageHeader].
func (e *GossipValidation) Marshal(buf []byte) error {
	return ErrMethodNotImplemented
}

// Returns the size of the GossipValidation packet.
func (e *GossipValidation) CalcSize() int {
	//FABIO deleted the boolean
	return binary.Size(e) - 1
}
