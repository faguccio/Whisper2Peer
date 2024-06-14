package verticalapi

import (
	"reflect"
	"testing"
)

func TestUnmarshalGossipAnnounce(t *testing.T) {
	result := GossipAnnounce{
		MessageHeader{3072, MessageType(500)},
		22,
		22,
		GossipType(17477),
		[]byte{18, 19, 20, 21},
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
