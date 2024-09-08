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

func TestMarshalGossipNotification(t *testing.T) {
	sample := GossipNotification{
		common.GossipNotification{
			MessageId: 5655,
			DataType:  common.GossipType(17477),
			Data:      []byte{18, 19, 20, 21},
		},
		MessageHeader{12, MessageType(502)},
	}

	wrongType := GossipNotification{
		common.GossipNotification{
			MessageId: 5655,
			DataType:  common.GossipType(17477),
			Data:      []byte{18, 19, 20, 21},
		},
		MessageHeader{12, MessageType(503)},
	}
	//In python list((integer).to_bytes(4, byteorder = 'big'))
	result := []byte{0, 12, 1, 246, 22, 23, 68, 69, 18, 19, 20, 21}
	var buf []byte
	// wrongType := []byte{132, 3, 1, 245, 2, 22, 23, 55, 43, 2}
	// smallBuf := []byte{132, 3, 1, 246, 22}

	_, err := wrongType.Marshal(buf)
	if err == nil {
		t.Fatalf("Marshal did not detect wrong message type")
	}

	// e.MessageHeader.Unmarshal(smallBuf)
	// _, err = e.Unmarshal(smallBuf)

	// if err != ErrNotEnoughData {
	// 	t.Fatalf("Unmarshal did not detect to small buffer")
	// }

	buf2, err := sample.Marshal(buf)
	if err != nil {
		t.Fatalf("Marshal threw an error on a valid input")
	}

	if !reflect.DeepEqual(result, buf2) {
		t.Fatal("Marshal result different than expected")
	}
}
