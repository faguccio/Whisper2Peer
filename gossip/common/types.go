package common

import "encoding/binary"

// This struct serves as collection of data needed to handle / communicate with
// a registered module
type RegisteredModule struct {
	MainToVert chan<- ToVert
}

// Type for the DataType of Gossip Messages.
type GossipType uint16

//go-sumtype:decl FromVert

// "union" with the types the verticalAPI might send
// TODO: this is not so nice since it somehow couples this module to the verticalAPI
type FromVert interface {
	isFromVert()
}

//go-sumtype:decl ToVert

// "union" with the types the verticalAPI needs to be able to receive
// TODO: this is not so nice since it somehow couples this module to the verticalAPI
type ToVert interface {
	isToVert()
}

//go-sumtype:decl ToStrat

// "union" with the types the gossip strategy needs to be able to receive
// TODO: this is not so nice since it somehow couples this module to the strats
type ToStrat interface {
	isToStrat()
}

//go-sumtype:decl FromStrat

// "union" with the types the gossip strategy might send
// TODO: this is not so nice since it somehow couples this module to the strats
type FromStrat interface {
	isFromStrat()
}

// This type represents a GossipAnnounce packet in the verticalApi.
type GossipAnnounce struct {
	TTL      uint8
	Reserved uint8
	DataType GossipType
	Data     []byte
}

// Returns the size of this struct.
func (e *GossipAnnounce) CalcSize() int {
	s := binary.Size(e.TTL)
	s += binary.Size(e.Reserved)
	s += binary.Size(e.DataType)
	s += len(e.Data)
	return s
}

// Mark this type as fromVert
func (e GossipAnnounce) isFromVert() {}

// Mark this type as toStrat
func (e GossipAnnounce) isToStrat() {}

// This type represents a GossipNotification packet in the verticalApi.
type GossipNotification struct {
	MessageId uint16
	DataType  GossipType
	Data      []byte
}

// Returns the size of this struct.
func (e *GossipNotification) CalcSize() int {
	s := binary.Size(e.MessageId)
	s += binary.Size(e.DataType)
	s += len(e.Data)
	return s
}

// Mark this type as toVert
func (e GossipNotification) isToVert() {}

// Mark this type as fromStrat
func (e GossipNotification) isFromStrat() {}

// This type represents a GossipNotify packet in the verticalApi.
type GossipNotify struct {
	Reserved uint16
	DataType GossipType
}

// Returns the size of this struct.
func (e *GossipNotify) CalcSize() int {
	return binary.Size(e)
}

// Wrapper for the GossipNotify message which also includes data about the registration
type GossipRegister struct {
	Data   GossipNotify
	Module *RegisteredModule
}

// Mark this type as fromVert
func (e GossipRegister) isFromVert() {}

// This type represents a GossipValidation packet in the verticalApi.
type GossipValidation struct {
	MessageId uint16
	Bitfield  uint16
	// only for ease of use we extract this from the bitfield on Unmarshal
	Valid bool
}

// Returns the size of this struct.
func (e *GossipValidation) CalcSize() int {
	s := binary.Size(e.MessageId)
	s += binary.Size(e.Bitfield)
	return s
}
func (e *GossipValidation) SetValid(v bool) {
	e.Valid = v
	if v {
		e.Bitfield = e.Bitfield | (uint16(1) << 0)
	} else {
		e.Bitfield = e.Bitfield & ^(uint16(1) << 0)
	}
}

// Mark this type as fromVert
func (e GossipValidation) isFromVert() {}

// Mark this type as toStrat
func (e GossipValidation) isToStrat() {}
