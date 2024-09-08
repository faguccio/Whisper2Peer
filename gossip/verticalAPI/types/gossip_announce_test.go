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
	"gossip/common"
	"reflect"
	"testing"
)

func TestUnmarshalGossipAnnounce(t *testing.T) {
	result := GossipAnnounce{
		common.GossipAnnounce{
			TTL:      22,
			Reserved: 22,
			DataType: common.GossipType(17477),
			Data:     []byte{18, 19, 20, 21},
		},
		MessageHeader{3072, MessageType(500)},
	}
	//In python list((integer).to_bytes(4, byteorder = 'big'))
	sample := []byte{12, 0, 1, 244, 22, 22, 68, 69, 18, 19, 20, 21}
	wrongType := []byte{12, 0, 1, 245, 2, 22, 18, 19, 20, 21}
	smallBuf := []byte{12, 0, 1, 244, 22}
	var e GossipAnnounce

	e.MessageHeader.Unmarshal(wrongType)
	_, err := e.Unmarshal(wrongType)
	if err == nil {
		t.Fatalf("Unmarshal did not detect wrong message type")
	}

	e.MessageHeader.Unmarshal(smallBuf)
	_, err = e.Unmarshal(smallBuf)

	if err != ErrNotEnoughData {
		t.Fatalf("Unmarshal did not detect to small buffer")
	}

	e.MessageHeader.Unmarshal(sample)
	_, err = e.Unmarshal(sample)

	if err != nil {
		t.Fatalf("Unmarshal threw an error on a valid input")
	}

	if !reflect.DeepEqual(result, e) {
		t.Fatal("Unmarshal result different than expected")
	}
}
