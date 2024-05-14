// Package verticalapi implements types for the packets used in the communication with other modules.
//
// Each packet includes/embedds the [MessageHeader]. Setting the correct values (for instance the size value stored in the MessageHeader) is completely up to the user.
// All packet types have the `CalcSize`, the `Marshal` and the `Unmarshal` functions (though `Marshal`/`Unmarshal` might not be implemented yet).
// Be aware that `Marshal`/`Unmarshal` are not completely each others counterpart.
// While `Marshal` also marshals the [MessageHeader], `Unmarshal` does not handle the [MessageHeader] (you might have already unmarshaled the Header to identify the message-type anyhow).
package verticalapi

import "errors"

// This file is like the root file for the verticalApi.

// Type for the DataType of Gossip Messages.
type GossipType uint16

// Definition of possible errors that might occur in this package.
var (
	ErrNotEnoughData    = errors.New("Not enough data")
	ErrBufSize          = errors.New("Provided buffer is too small")
	ErrWrongMessageType = errors.New("Wrong MessageType set in header")
	// usually only used to signal that Marshal/Unmarshal is not provided by that message type
	ErrMethodNotImplemented = errors.New("Method is not implemented by that specific message type")
)

// Type for MessageType set in [MessageHeader].
type MessageType uint16

// Constants for the [MessageType]
const (
	// MessageType for the [GossipAnnounce] packet.
	GossipAnnounceType = 500
	// MessageType for the [GossipNotification] packet.
	GossipNotificationType = 502
	// MessageType for the [GossipNotify] packet.
	GossipNotifyType = 501
	// MessageType for the [GossipValidation] packet.
	GossipValidationType = 503
)
