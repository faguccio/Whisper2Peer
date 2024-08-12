package pow

import (
	"encoding/binary"
	"io"
	"slices"
)

type TM struct {
	header uint16
	data [16]byte
	nonce uint16
}

func (x *TM) Marshal(buf []byte) ([]byte, error) {
	buf = slices.Grow(buf, binary.Size(x.header) + len(x.data) + binary.Size(x.nonce))
	buf = buf[:binary.Size(x.header) + len(x.data) + binary.Size(x.nonce)]
	binary.BigEndian.AppendUint16(buf, x.header)
	copy(buf[binary.Size(x.header):], x.data[:])
	binary.BigEndian.AppendUint16(buf, x.nonce)
	return buf, nil
}

func (x *TM) Nonce() uint16 {
	return x.nonce
}

func (x *TM) SetNonce(n uint16) {
	x.nonce = n
}

func (x *TM) AddToNonce(n uint16) {
	x.nonce += n
}

func (x *TM) StripPrefixLen() uint {
	return uint(binary.Size(x.header))
}

func (x *TM) PrefixLen() uint {
	return uint(binary.Size(x.data))
}

func (x *TM) WriteNonce(w io.Writer) {
	w.Write(binary.BigEndian.AppendUint16(nil, x.nonce))
}
