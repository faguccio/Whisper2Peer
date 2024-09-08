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

package verticalapi

import (
	"encoding/binary"
	"errors"
	"gossip/common"
	"slices"
)

// This type represents a GossipNotification packet in the verticalApi.
type GossipNotification struct {
	Gn            common.GossipNotification
	MessageHeader MessageHeader
}

// // Unmarshals the GossipNotification packet from the provided buffer.
// //
// // Returns the number of bytes read from the buffer.
func (e *GossipNotification) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != GossipNotificationType {
		return 0, errors.New("wrong type")
	}

	if len(buf) < e.CalcSize() {
		return 0, ErrNotEnoughData
	}

	idx := e.MessageHeader.CalcSize()

	e.Gn.MessageId = binary.BigEndian.Uint16(buf[idx:])
	idx += 2

	e.Gn.DataType = common.GossipType(binary.BigEndian.Uint16(buf[idx:]))
	idx += 2

	// golang slices: [a:b] index b is excluded
	e.Gn.Data = buf[idx:min(int(e.MessageHeader.Size), len(buf))]
	idx += len(e.Gn.Data)

	return idx, nil
}

// Marshals the GossipNotification packet to the provided buffer.
//
// If the provided buffer is too small, this function will just grow it.
func (e *GossipNotification) Marshal(buf []byte) ([]byte, error) {
	if e.MessageHeader.Type != GossipNotificationType {
		return nil, errors.New("wrong type")
	}

	buf = slices.Grow(buf, e.CalcSize())
	buf = buf[:e.CalcSize()]

	if err := e.MessageHeader.Marshal(buf); err != nil {
		return nil, err
	}

	idx := e.MessageHeader.CalcSize()

	binary.BigEndian.PutUint16(buf[idx:], e.Gn.MessageId)
	idx += 2

	binary.BigEndian.PutUint16(buf[idx:], uint16(e.Gn.DataType))
	idx += 2

	copy(buf[idx:], e.Gn.Data)
	idx += len(e.Gn.Data)

	return buf, nil
}

// Returns the size of the GossipNotification packet.
func (e *GossipNotification) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.Gn.MessageId)
	s += binary.Size(e.Gn.DataType)
	s += len(e.Gn.Data)
	return s
}

// Mark this type as vertical type
func (e *GossipNotification) isVertType() {}
