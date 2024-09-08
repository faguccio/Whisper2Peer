/*
 * gossip
 * Copyright (C) 2024 Fabio Gaiba and Lukas Heindl
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Package verticalapi implements types for the packets used in the communication with other modules.
//
// Each packet includes/embedds the [MessageHeader]. Setting the correct values (for instance the size value stored in the MessageHeader) is completely up to the user.
// All packet types have the `CalcSize`, the `Marshal` and the `Unmarshal` functions (though `Marshal`/`Unmarshal` might not be implemented yet).
// Be aware that `Marshal`/`Unmarshal` are not completely each others counterpart.
// While `Marshal` also marshals the [MessageHeader], `Unmarshal` does not handle the [MessageHeader] (you might have already unmarshaled the Header to identify the message-type anyhow).
package verticalapi

import "errors"

// This file is like the root file for the verticalApi.

// Definition of possible errors that might occur in this package.
var (
	ErrNotEnoughData    = errors.New("not enough data")
	ErrBufSize          = errors.New("provided buffer is too small")
	ErrWrongMessageType = errors.New("wrong MessageType set in header")
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

type VertType interface {
	CalcSize() int
	isVertType()
}
