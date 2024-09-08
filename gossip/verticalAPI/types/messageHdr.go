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
)

// This type represents the MessageHeader in the verticalApi.
type MessageHeader struct {
	Size uint16
	Type MessageType
}

// Unmarshals the MessageHeader from the provided buffer.
//
// Returns the number of bytes read from the buffer.
func (m *MessageHeader) Unmarshal(buf []byte) (int, error) {
	if len(buf) < m.CalcSize() {
		return 0, ErrNotEnoughData
	}

	idx := 0

	m.Size = binary.BigEndian.Uint16(buf[0:])
	idx += 2

	m.Type = MessageType(binary.BigEndian.Uint16(buf[idx:]))
	idx += 2

	return idx, nil
}

// Marshals the MessageHeader to the provided buffer.
//
// This function expects that the provided buffer already is large enough.
func (m *MessageHeader) Marshal(buf []byte) error {
	if len(buf) < m.CalcSize() {
		return ErrBufSize
	}
	binary.BigEndian.PutUint16(buf, m.Size)
	binary.BigEndian.PutUint16(buf[2:], uint16(m.Type))
	return nil
}

// Returns the size of the MessageHeader.
func (m *MessageHeader) CalcSize() int {
	return binary.Size(m)
}

func (m *MessageHeader) RecalcSize(e VertType) {
	m.Size = uint16(e.CalcSize())
}
