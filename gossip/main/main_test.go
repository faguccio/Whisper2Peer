package gossip_test

import (
	"context"
	"gossip/common"
	"gossip/internal/testutils"
	gossip "gossip/main"
	vtypes "gossip/verticalAPI/types"
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
	// Marshal(buf []byte) error
	// CalcSize() int
}

// collection of msgType and the associated buffer
// needed because these types do not implement unmarshal at the moment
type tester struct {
	msg  msgType
	buf  []byte
	name string
}

func NotTestMainHorizontal(test *testing.T) {
	var testLog *slog.Logger = slogt.New(test)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"placeholder", "-gossip", "1", "-cache", "2", "-peers", "127.0.1.1:6001,127.0.2.1:6001"}

	m := gossip.NewMain()

	// run
	initFin := make(chan error, 1)
	go m.Run(initFin)
	defer m.Close()
	err := <-initFin
	if err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)
	testLog.Debug("Endtest")

}

// Test main. This is not a proper Unit Testing, more of a simulation for me to actually see how the main is working
func NotTestMain(test *testing.T) {
	var testLog *slog.Logger = slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}))

	// use table driven testing: https://go.dev/wiki/TableDrivenTests
	// define the messages that should be received via the socket
	ts := []tester{
		{
			msg: common.GossipAnnounce{
				TTL:      32,
				Reserved: 0,
				DataType: 24,
				Data:     []byte{0x20, 0x50},
			},
			buf:  []byte{0x0, 0x0a, 0x01, 0xf4, 32, 0, 0x0, 0x18, 0x20, 0x50},
			name: "announce",
		},
		{
			msg: common.GossipNotify{
				Reserved: 0,
				DataType: 42,
			},
			buf:  []byte{0x0, 0x08, 0x01, 0xf5, 0, 0, 0x0, 0x2a},
			name: "notify",
		},
		{
			msg: common.GossipAnnounce{
				TTL:      32,
				Reserved: 0,
				DataType: 42,
				Data:     []byte{0x20, 0x50},
			},
			buf:  []byte{0x0, 0x0a, 0x01, 0xf4, 32, 0, 0x0, 0x2a, 0x20, 0x50},
			name: "announce",
		},
		{
			msg: common.GossipValidation{
				MessageId: 1337,
				// setting to 0b..1 does not work since the valid flag is not
				// imported (-> or do not use reflect.DeepEqual later)
				Bitfield: 0,
			},
			buf:  []byte{0x0, 0x08, 0x01, 0xf7, 0x05, 0x39, 0, 0},
			name: "validation",
		},
	}

	m := gossip.NewMain()

	// run
	initFin := make(chan error, 1)
	go m.Run(initFin)
	defer m.Close()
	err := <-initFin
	if err != nil {
		panic(err)
	}

	testLog.Debug("START MAIN TRY (NOT A REAL TEST)")

	conn, err := net.Dial("tcp", "localhost:13379")
	if err != nil {
		testLog.Error("Error connecting, should not happen", "error", err)
		return
	}
	defer conn.Close()

	for _, t := range ts {

		_, err = conn.Write(t.buf)
		if err != nil {
			testLog.Error("Error sending data", "error", err)
			return
		}
	}

	time.Sleep(1 * time.Second)
}

func TestMainEndToEndOneHopA(test *testing.T) {
	var testLog *slog.Logger = slogt.New(test)
	t, err := testutils.NewTesterFromJSON("../test_assets/e2e.json")
	if err != nil {
		panic(err)
	}
	err = t.AddLogger(testLog)
	if err != nil {
		panic(err)
	}
	if err = t.Startup("127.0.0.1"); err != nil {
		panic(err)
	}
	if err = t.RegisterAllPeersForType(1337); err != nil {
		panic(err)
	}
	p := t.Peers[0]
	msg := vtypes.GossipAnnounce{
		Ga: common.GossipAnnounce{
			TTL:      2,
			Reserved: 0,
			DataType: 1337,
			Data:     []byte{1},
		},
		MessageHeader: vtypes.MessageHeader{
			Type: vtypes.GossipAnnounceType,
		},
	}
	msg.MessageHeader.RecalcSize(&msg)
	if err = p.SendMsg(&msg); err != nil {
		panic(err)
	}

	ctx, cfunc := context.WithTimeout(context.Background(), time.Minute)
	defer cfunc()
	// interval is two gossip rounds long
	err = t.WaitUntilSilent(ctx, true, 0, 2*time.Second)
	if err != nil {
		test.Fatalf("wait exited with %v", err)
	}
	time.Sleep(1 * time.Second)

	t.Teardown()

	if _, data, err := t.ProcessReachedDistCnt(0, 0, true); err == nil {
		if len(data) != 4 {
			test.Fatalf("something went wrong more than %d distinct distances are registered: %v", 4, data)
		}

		if data[0] != 1 {
			test.Fatalf("message was received by %d nodes with distance 0 (should be %d nodes)", data[0], 1)
		}
		if data[1] != 1 {
			test.Fatalf("message was received by %d nodes with distance 1 (should be %d nodes)", data[1], 1)
		}
		if data[2] != 1 {
			test.Fatalf("message was received by %d nodes with distance 2 (should be %d nodes)", data[2], 1)
		}
		if data[3] != 0 {
			test.Fatalf("message was received by %d nodes with distance 3 (should be %d nodes)", data[3], 1)
		}
	} else {
		panic(data)
	}
}

func TestMainEndToEndOneHopB(test *testing.T) {
	var testLog *slog.Logger = slogt.New(test)
	t, err := testutils.NewTesterFromJSON("../test_assets/e2e.json")
	if err != nil {
		panic(err)
	}
	err = t.AddLogger(testLog)
	if err != nil {
		panic(err)
	}
	if err = t.Startup("127.0.1.1"); err != nil {
		panic(err)
	}
	if err = t.RegisterAllPeersForType(1337); err != nil {
		panic(err)
	}
	p := t.Peers[0]
	msg := vtypes.GossipAnnounce{
		Ga: common.GossipAnnounce{
			TTL:      0,
			Reserved: 0,
			DataType: 1337,
			Data:     []byte{1},
		},
		MessageHeader: vtypes.MessageHeader{
			Type: vtypes.GossipAnnounceType,
		},
	}
	msg.MessageHeader.RecalcSize(&msg)
	if err = p.SendMsg(&msg); err != nil {
		panic(err)
	}

	ctx, cfunc := context.WithTimeout(context.Background(), time.Minute)
	defer cfunc()
	// interval is two gossip rounds long
	err = t.WaitUntilSilent(ctx, true, 0, 2*time.Second)
	if err != nil {
		test.Fatalf("wait exited with %v", err)
	}
	time.Sleep(1 * time.Second)

	t.Teardown()

	if _, data, err := t.ProcessReachedDistCnt(0, 0, true); err == nil {
		if len(data) != 4 {
			test.Fatalf("something went wrong more than %d distinct distances are registered: %v", 4, data)
		}

		if data[0] != 1 {
			test.Fatalf("message was received by %d nodes with distance 0 (should be %d nodes)", data[0], 1)
		}
		if data[1] != 1 {
			test.Fatalf("message was received by %d nodes with distance 1 (should be %d nodes)", data[1], 1)
		}
		if data[2] != 1 {
			test.Fatalf("message was received by %d nodes with distance 2 (should be %d nodes)", data[2], 1)
		}
		if data[3] != 1 {
			test.Fatalf("message was received by %d nodes with distance 3 (should be %d nodes)", data[3], 1)
		}
	} else {
		panic(data)
	}
}

func TestMainEndToEndOneHopC(test *testing.T) {
	var testLog *slog.Logger = slogt.New(test)
	t, err := testutils.NewTesterFromJSON("../test_assets/erdos.json")
	if err != nil {
		panic(err)
	}
	err = t.AddLogger(testLog)
	if err != nil {
		panic(err)
	}
	if err = t.Startup("127.0.2.1"); err != nil {
		panic(err)
	}
	if err = t.RegisterAllPeersForType(1337); err != nil {
		panic(err)
	}
	p := t.Peers[0]
	msg := vtypes.GossipAnnounce{
		Ga: common.GossipAnnounce{
			TTL:      10,
			Reserved: 0,
			DataType: 1337,
			Data:     []byte{1},
		},
		MessageHeader: vtypes.MessageHeader{
			Type: vtypes.GossipAnnounceType,
		},
	}
	msg.MessageHeader.RecalcSize(&msg)
	if err = p.SendMsg(&msg); err != nil {
		panic(err)
	}

	ctx, cfunc := context.WithTimeout(context.Background(), time.Minute)
	defer cfunc()
	// interval is two gossip rounds long
	err = t.WaitUntilSilent(ctx, true, 0, 2*time.Second)
	if err != nil {
		test.Fatalf("wait exited with %v", err)
	}
	time.Sleep(1 * time.Second)

	t.Teardown()

	data, err := t.ProcessReachedWhen(common.GossipType(1337), true)
	// Just checking all nodes have received the message
	if err != nil {
		panic(err)
	}

	if len(data) != 20 {
		test.Fatalf("message was received by %d nodes (should be %d nodes)", len(data), 20)
	}
}
