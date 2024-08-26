package strats

import (
	"crypto/rand"
	horizontalapi "gossip/horizontalAPI"
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
	payload := cookie.createCookie(aead)
	readCookie, err := ReadCookie(aead, payload)

	if err != nil {
		test.Fatalf("Decryption of cookie failed")
	}

	if reflect.DeepEqual(readCookie.chall, cookie.chall) {
		test.Fatalf("Read nonce different from nonce (%d, %d)", readCookie.chall, cookie.chall)
	}

	// if readCookie.timestamp != cookie.timestamp {
	// 	test.Fatalf("Read date different from initial one (%v, %v)", readCookie.timestamp, cookie.timestamp)
	// }

	if string(readCookie.dest) != string(cookie.dest) {
		test.Fatalf("Read dest different from dest (%s, %s)", readCookie.dest, cookie.dest)
	}

}
