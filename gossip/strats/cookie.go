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

// This struct represents the connection cookie.
type connCookie struct {
	// The cipher Nonce
	chall     []byte
	timestamp time.Time
	dest      horizontalapi.ConnectionId
}

// Return a new cookie object with provided destination (timestamp is set to now)
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

// This function will marshal the cookie object and return the resulting byte slice
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

// This function takes a buffer (marshalled cookie) and will unmarshal the data
// in the correct fields of the caller
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

// Marshall the cookie, encrypt it and add Nonce. Return the resulting byte slice
func (x *connCookie) CreateCookie(aead cipher.AEAD) []byte {
	payload := x.Marshal()

	ciphertext := aead.Seal(nil, x.chall, payload, nil)
	// Cipher nonce is appended as the first (24) bytes of the cookie
	cookie := slices.Concat(x.chall, ciphertext)
	return cookie
}

// This function takes a byte slice (cookie) and return a nonce for the PoW
//
// The computed nonce will have the hash starting with 8 zeros
func ComputePoW(cookie []byte) uint64 {
	mypow := powMarsh{
		PowNonce: 0,
		Cookie:   cookie,
	}

	nonce := pow.ProofOfWork(func(digest []byte) bool {
		return pow.First8bits0(digest)
	}, &mypow)

	return nonce
}

// This function takes an the ChaCha20 cipher and a marshalled cookie. It will
//
// If the decryption fail it will return an error, other wise, the cookie object
func ReadCookie(aead cipher.AEAD, cookie []byte) (*connCookie, error) {
	cipherNonce := cookie[0:chacha20poly1305.NonceSizeX]
	cookie = cookie[chacha20poly1305.NonceSizeX:]

	plaintext, err := aead.Open(nil, cipherNonce, cookie, nil)
	if err != nil {
		return nil, fmt.Errorf("Error while decrypting the cookie %w", err)
	}

	var c connCookie
	c.Unmarshal(plaintext)
	return &c, nil
}
