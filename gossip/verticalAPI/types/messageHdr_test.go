package verticalapi

import (
	"bytes"
	"fmt"
	"testing"
)

func TestUnmarshalHeader(t *testing.T) {
	sample := []byte{132, 3, 234, 69}
	// results generated using python: int.from_bytes([234, 69], "big")

	result := MessageHeader{33795, MessageType(59973)}

	var m MessageHeader

	small_buf := []byte{0, 0, 0}
	_, err := m.Unmarshal(small_buf)

	if err == nil {
		t.Fatalf("Unmarshal failed to throw error on not big enough buffer")
	}

	_, err = m.Unmarshal(sample)
	if err != nil {
		fmt.Println(err)
		t.Fatalf("Unmarshal error")
	}

	if m != result {
		t.Fatalf("Unmarshal result different than expected")
	}
}

func TestMarshalHeader(t *testing.T) {
	m := MessageHeader{33795, MessageType(59973)}
	result := []byte{132, 3, 234, 69}

	small_buf := []byte{0, 1, 2}
	err := m.Marshal(small_buf)

	if err == nil {
		t.Fatalf("Marshal failed to throw error on not big enough buffer")
	}

	buf := []byte{0, 0, 0, 0}
	err = m.Marshal(buf)
	if err != nil {
		t.Fatalf("Marshal errored despite large enough buffer. err: %v", err)
	}

	if !bytes.Equal(buf, result) {
		t.Fatalf("Marshal result different than expected")
	}
}

// func (m *MessageHeader) Marshal(buf []byte) error {
// 	if len(buf) < m.CalcSize() {
// 		return ErrBufSize
// 	}
// 	binary.BigEndian.PutUint16(buf, m.Size)
// 	binary.BigEndian.PutUint16(buf[2:], uint16(m.Type))
// 	return nil
// }
