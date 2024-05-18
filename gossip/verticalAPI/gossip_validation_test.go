package verticalapi

import (
	"fmt"
	"testing"
)

func TestUnmarshalGossipVal(t *testing.T) {
	result := GossipValidation{
		MessageHeader{33795, MessageType(503)},
		31543,
		17477,
		// only for ease of use we extract this from the bitfield on Unmarshal
		true,
	}
	//In python list((integer).to_bytes(4, byteorder = 'big'))
	sample := []byte{132, 3, 1, 247, 123, 55, 68, 69}
	wrongType := []byte{132, 3, 2, 247, 123, 55, 43, 2}
	smallBuf := []byte{132, 3, 1, 247, 123, 55, 43}
	var e GossipValidation
	//e.MessageHeader = MessageHeader{33795, MessageType(2)}

	_, err := e.Unmarshal(wrongType)
	if err == nil {
		t.Fatalf("Unmarshal did not detect wrong message type")
	}

	e.MessageHeader = MessageHeader{33795, MessageType(503)}
	_, err = e.Unmarshal(smallBuf)

	if err == nil {
		t.Fatalf("Unmarshal did not detect to small buffer")
	}

	_, err = e.Unmarshal(sample)

	if err != nil {
		t.Fatalf("Unmarshal threw an error on a valid input")
	}

	fmt.Println(e, result)
	if result != e {
		t.Fatal("Unmarshal result different than expected")
	}

	// small_buf := []byte{0, 0, 0}
	// _, err := m.Unmarshal(small_buf)

	// if err == nil {
	// 	t.Fatalf("Unmarshal failed to throw error on not big enough buffer")
	// }

	// _, err = m.Unmarshal(sample)
	// if err != nil {
	// 	fmt.Println(err)
	// 	t.Fatalf("Unmarshal error")
	// }

	// if m != result {
	// 	t.Fatalf("Unmarshal result different than expected")
	// }
}
