package strats

import (
	"fmt"
	horizontalapi "gossip/horizontalAPI"
	"testing"
	"time"
)

func TestCookie(test *testing.T) {

	nonce := uint64(12340987)
	date := time.Unix(0, time.Now().UnixMicro())
	dest := "miamibeach"

	cookie := connCookie{
		chall:     nonce,
		timestamp: date,
		dest:      horizontalapi.ConnectionId(dest),
	}

	payload := cookie.createCookie()

	var readCookie connCookie
	readCookie.readCookie(payload, nonce)
	fmt.Println(readCookie)

	if readCookie.chall != nonce {
		test.Fatalf("Read nonce different from nonce (%d, %d)", readCookie.chall, nonce)
	}

	if readCookie.timestamp != date {
		test.Fatalf("Read date different from initial one (%v, %v)", readCookie.timestamp, date)
	}

	if string(readCookie.dest) != dest {
		test.Fatalf("Read dest different from dest (%s, %s)", readCookie.dest, dest)
	}

}
