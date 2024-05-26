package main

import (
	"fmt"
	vertTypes "gossip/verticalAPI/types"
	"net"
	"testing"
	"time"

	_ "gopkg.in/ini.v1"
)

// abstraction for message types
// everything implementing below methods is considered a msgType
type msgType interface {
	Marshal(buf []byte) error
	CalcSize() int
}

// collection of msgType and the associated buffer
// needed because these types do not implement unmarshal at the moment
type tester struct {
	msg  msgType
	buf  []byte
	name string
}

// Test main
func TestMain(test *testing.T) {
	// use table driven testing: https://go.dev/wiki/TableDrivenTests
	// define the messages that should be received via the socket
	ts := []tester{
		{
			&vertTypes.GossipAnnounce{
				MessageHeader: vertTypes.MessageHeader{
					Size: 8 + 2,
					Type: vertTypes.GossipAnnounceType,
				},
				TTL:      32,
				Reserved: 0,
				DataType: 24,
				Data:     []byte{0x20, 0x50},
			},
			[]byte{0x0, 0x0a, 0x01, 0xf4, 32, 0, 0x0, 0x18, 0x20, 0x50},
			"announce",
		},
		{
			&vertTypes.GossipNotify{
				MessageHeader: vertTypes.MessageHeader{
					Size: 8,
					Type: vertTypes.GossipNotifyType,
				},
				Reserved: 0,
				DataType: 42,
			},
			[]byte{0x0, 0x08, 0x01, 0xf5, 0, 0, 0x0, 0x2a},
			"notify",
		},
		{
			&vertTypes.GossipValidation{
				MessageHeader: vertTypes.MessageHeader{
					Size: 8,
					Type: vertTypes.GossipValidationType,
				},
				MessageId: 1337,
				// setting to 0b..1 does not work since the valid flag is not
				// imported (-> or do not use reflect.DeepEqual later)
				Bitfield: 0,
			},
			[]byte{0x0, 0x08, 0x01, 0xf7, 0x05, 0x39, 0, 0},
			"validation",
		},
	}

	go main()
	fmt.Println("CIAO")

	conn, err := net.Dial("tcp", "localhost:13379")
	if err != nil {
		fmt.Println("Error connecting, should not happen:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(ts[0].buf)
	if err != nil {
		fmt.Println("Error sending data:", err)
		return
	}

	time.Sleep(3 * time.Second)
}
