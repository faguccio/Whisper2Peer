package main

import (
	vertTypes "gossip/verticalAPI/types"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/lmittmann/tint"
	"github.com/neilotoole/slogt"
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

func TestMainHorizontal(test *testing.T) {
	var testLog *slog.Logger = slogt.New(test)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"placeholder", "-gossip", "1", "-cache", "2", "-peers", "127.0.1.1:6001,127.0.2.1:6001"}
	go main()
	time.Sleep(2 * time.Second)
	testLog.Debug("Endtest")

}

// Test main. This is not a proper Unit Testing, more of a simulation for me to actually see how the main is working
func TestMain(test *testing.T) {
	var testLog *slog.Logger = slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}))

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
			&vertTypes.GossipAnnounce{
				MessageHeader: vertTypes.MessageHeader{
					Size: 8 + 2,
					Type: vertTypes.GossipAnnounceType,
				},
				TTL:      32,
				Reserved: 0,
				DataType: 42,
				Data:     []byte{0x20, 0x50},
			},
			[]byte{0x0, 0x0a, 0x01, 0xf4, 32, 0, 0x0, 0x2a, 0x20, 0x50},
			"announce",
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
	testLog.Debug("START MAIN TRY (NOT A REAL TEST)")

	conn, err := net.Dial("tcp", "localhost:13379")
	if err != nil {
		testLog.Error("Error connecting, should not happen:", err)
		return
	}
	defer conn.Close()

	for _, t := range ts {

		_, err = conn.Write(t.buf)
		if err != nil {
			testLog.Error("Error sending data:", err)
			return
		}
	}

	time.Sleep(1 * time.Second)
}
