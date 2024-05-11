package verticalapi

import (
	"encoding/binary"
)

type MessageHeader struct {
	Size uint16
	Type uint16
}

func (m *MessageHeader) Unmarshal(buf []byte) (int, error) {
	m.Size = binary.BigEndian.Uint16(buf)
	m.Type = binary.BigEndian.Uint16(buf[2:])
	return 32, nil
}
func (m *MessageHeader) Marshal(buf []byte) error {
	binary.BigEndian.PutUint16(buf, m.Size)
	binary.BigEndian.PutUint16(buf[2:], m.Type)
	return nil
}
func (m *MessageHeader) CalcSize() int {
	return binary.Size(m)
}
