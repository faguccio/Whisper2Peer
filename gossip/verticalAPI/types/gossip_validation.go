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

// Marshals the GossipValidation packet to the provided buffer.
func (e *GossipValidation) Marshal(buf []byte) ([]byte, error) {
	if e.MessageHeader.Type != GossipValidationType {
		return nil, errors.New("wrong type")
	}

	buf = slices.Grow(buf, e.CalcSize())
	buf = buf[:e.CalcSize()]

	if err := e.MessageHeader.Marshal(buf); err != nil {
		return nil, err
	}

	idx := e.MessageHeader.CalcSize()

	binary.BigEndian.PutUint16(buf[idx:], e.Gv.MessageId)
	idx += 2

	binary.BigEndian.PutUint16(buf[idx:], e.Gv.Bitfield)
	idx += 2

	return buf, nil
}

// Returns the size of the GossipValidation packet.
func (e *GossipValidation) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.Gv.MessageId)
	s += binary.Size(e.Gv.Bitfield)
	return s
}

// Mark this type as vertical type
func (e *GossipValidation) isVertType() {}
