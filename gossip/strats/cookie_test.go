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
	"crypto/rand"
	horizontalapi "gossip/horizontalAPI"
	pow "gossip/pow"
	"reflect"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"
)

func TestCookie(test *testing.T) {

	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		panic(err)
	}

	cookie := NewConnCookie(horizontalapi.ConnectionId("MIAMIbeach"))
	payload := cookie.CreateCookie(aead)
	readCookie, err := ReadCookie(aead, payload)

	if err != nil {
		test.Fatalf("Decryption of cookie failed")
	}

	if !reflect.DeepEqual(readCookie.chall, cookie.chall) {
		test.Fatalf("Read nonce different from nonce (%d, %d)", readCookie.chall, cookie.chall)
	}

	if readCookie.timestamp != cookie.timestamp {
		test.Fatalf("Read date different from initial one (%v, %v)", readCookie.timestamp, cookie.timestamp)
	}

	if string(readCookie.dest) != string(cookie.dest) {
		test.Fatalf("Read dest different from dest (%s, %s)", readCookie.dest, cookie.dest)
	}

	nonce := ComputePoW(payload)
	mypow := powMarsh{PowNonce: nonce, Cookie: payload}

	powValidity := pow.CheckProofOfWork(func(digest []byte) bool {
		return pow.First8bits0(digest)
	}, &mypow)

	if !powValidity {
		test.Fatalf("Computing or checking the proof of work leads to wrong result")
	}

}
