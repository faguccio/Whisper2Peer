package strats

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	horizontalapi "gossip/horizontalAPI"
	pow "gossip/pow"
	"slices"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

type connCookie struct {
	chall     []byte
	timestamp time.Time
	dest      horizontalapi.ConnectionId
}

// Return a cookie object with provided destination
func NewConnCookie(dest horizontalapi.ConnectionId) connCookie {
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}

	return connCookie{
		chall:     nonce,
		timestamp: time.Unix(0, time.Now().UnixNano()),
		dest:      dest,
	}
}

func (x *connCookie) Marshal() []byte {
	timestamp := x.timestamp.UnixNano()
	buf := make([]byte, len(x.chall)+binary.Size(timestamp)+len(x.dest))

	idx := 0

	binary.BigEndian.PutUint64(buf[idx:], uint64(timestamp))
	idx += binary.Size(timestamp)

	copy(buf[idx:], x.chall[:])
	idx += len(x.chall)

	copy(buf[idx:], x.dest[:])
	idx += len(x.dest)

	return buf
}

func (x *connCookie) Unmarshal(buf []byte) {
	idx := 0

	t := binary.BigEndian.Uint64(buf[idx : idx+8])
	timestamp := time.Unix(0, int64(t))
	idx += 8

	x.chall = make([]byte, chacha20poly1305.NonceSizeX)
	copy(x.chall, buf[idx:idx+chacha20poly1305.NonceSizeX])
	idx += chacha20poly1305.NonceSizeX

	dest := make([]byte, len(buf[idx:]))
	copy(dest, buf[idx:])

	x.timestamp = timestamp
	x.dest = horizontalapi.ConnectionId(string(dest))
}

func (x *connCookie) CreateCookie(aead cipher.AEAD) []byte {
	payload := x.Marshal()

	ciphertext := aead.Seal(nil, x.chall, payload, nil)
	// Cipher nonce is appended as the first (24) bytes of the cookie
	cookie := slices.Concat(x.chall, ciphertext)
	return cookie
}

func ComputePoW(cookie []byte) horizontalapi.ConnPoW {
	mypow := powMarsh{
		PowNonce: 0,
		Cookie:   cookie,
	}

	nonce := pow.ProofOfWork(func(digest []byte) bool {
		return pow.First8bits0(digest)
	}, &mypow)

	mypow.PowNonce = nonce
	return horizontalapi.ConnPoW{PowNonce: mypow.PowNonce, Cookie: mypow.Cookie}
}

func ReadCookie(aead cipher.AEAD, cookie []byte) (*connCookie, error) {
	cipherNonce := cookie[0:chacha20poly1305.NonceSizeX]
	cookie = cookie[chacha20poly1305.NonceSizeX:]

	plaintext, err := aead.Open(nil, cipherNonce, cookie, nil)
	if err != nil {
		fmt.Println("Error while decrypting the cookie", "err", err)
		return nil, err
	}

	var c connCookie
	c.Unmarshal(plaintext)
	return &c, nil
}
