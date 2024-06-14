package verticalapi

import (
	"reflect"
	"testing"
)

func TestMarshalGossipNotification(t *testing.T) {
	sample := GossipNotification{
		MessageHeader{12, MessageType(502)},
		5655,
		GossipType(17477),
		[]byte{18, 19, 20, 21},
	}

	wrongType := GossipNotification{
		MessageHeader{12, MessageType(503)},
		5655,
		GossipType(17477),
		[]byte{18, 19, 20, 21},
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
