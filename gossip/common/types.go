package common

import "context"

// generic identifier used for a connection
type ConnectionId string

// store arbitrary data along with the connection it belongs to
type Conn[T any] struct {
	Id    ConnectionId
	Data  T
	Ctx   context.Context
	Cfunc context.CancelFunc
}

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

// Mark this type as toVert
func (e GossipNotification) isToVert() {}

// Mark this type as fromStrat
func (e GossipNotification) isFromStrat() {}

// This type represents a GossipNotify packet in the verticalApi.
type GossipNotify struct {
	Reserved uint16
	DataType GossipType
}

// Wrapper for the GossipNotify message which also includes data about the registration
type GossipRegister struct {
	Data   GossipNotify
	Module *Conn[RegisteredModule]
}

// Mark this type as fromVert
func (e GossipRegister) isFromVert() {}

type GossipUnRegister ConnectionId

// Mark this type as fromVert
func (e GossipUnRegister) isFromVert() {}

// This type represents a GossipValidation packet in the verticalApi.
type GossipValidation struct {
	MessageId uint16
	Bitfield  uint16
	// only for ease of use we extract this from the bitfield on Unmarshal
	Valid bool
}

// Convenience function to set the valid flag on this message (sets .valid and adjusts the bitfield)
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
