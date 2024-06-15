package verticalapi

import (
	"gossip/common"
	"testing"
)

func TestUnmarshalGossipNotify(t *testing.T) {
	result := GossipNotify{
		common.GossipNotify{
			Reserved: 31543,
			DataType: common.GossipType(17477),
		},
		MessageHeader{33795, MessageType(501)},
	}
	//In python list((integer).to_bytes(4, byteorder = 'big'))
	sample := []byte{132, 3, 1, 245, 123, 55, 68, 69}
	wrongType := []byte{132, 3, 2, 247, 123, 55, 43, 2}
	smallBuf := []byte{132, 3, 1, 245, 123, 55, 43}
	var e GossipNotify

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

	if result != e {
		t.Fatal("Unmarshal result different than expected")
	}
}
