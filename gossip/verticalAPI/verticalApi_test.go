package verticalapi

import (
	"errors"
	vertTypes "gossip/verticalAPI/types"
	"io"
	"log/slog"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/neilotoole/slogt"
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

func TestHandleConnection(test *testing.T) {
	test.Parallel()
	// use table driven testing: https://go.dev/wiki/TableDrivenTests
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
	for _, t := range ts {
		t := t // NOTE: /wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		test.Run(t.name, func(test *testing.T) {
			test.Parallel()
			var testLog *slog.Logger = slogt.New(test)
			cVert, cTest := net.Pipe()
			vertToMainChans := VertToMainChans{
				Register:   make(chan VertToMainRegister, 1),
				Anounce:    make(chan VertToMainAnnounce, 1),
				Validation: make(chan VertToMainValidation, 1),
			}
			vert := NewVerticalApi(testLog, vertToMainChans)

			vert.conns[cVert] = struct{}{}
			mainToVert := make(chan MainToVertNotification)
			regMod := RegisteredModule{
				MainToVert: mainToVert,
			}
			go vert.handleConnection(cVert, regMod)

			if n, err := cTest.Write(t.buf); err != nil {
				test.Fatalf("failed sending: %v %+v", err, t)
			} else if n != len(t.buf) {
				test.Fatalf("buffer wasn't sent completely (might need to adjust the test) %v", t)
			}

			time.Sleep(5 * time.Second)

			select {
			case x := <-vertToMainChans.Validation:
				t, ok := t.msg.(*vertTypes.GossipValidation)
				if !ok {
					test.Fatalf("handler sent to wrong channel")
				}
				if !reflect.DeepEqual(x.Data, *t) {
					test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x.Data, *t)
				}
			case x := <-vertToMainChans.Anounce:
				t, ok := t.msg.(*vertTypes.GossipAnnounce)
				if !ok {
					test.Fatalf("handler sent to wrong channel")
				}
				if !reflect.DeepEqual(x.Data, *t) {
					test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x.Data, *t)
				}
			case x := <-vertToMainChans.Register:
				t, ok := t.msg.(*vertTypes.GossipNotify)
				if !ok {
					test.Fatalf("handler sent to wrong channel")
				}
				if !reflect.DeepEqual(x.Data, *t) {
					test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x.Data, *t)
				}
			default:
				test.Fatalf("handler didn't pass a message to any vertToMain channel")
			}
			if len(vertToMainChans.Anounce) != 0 || len(vertToMainChans.Register) != 0 || len(vertToMainChans.Validation) != 0 {
				test.Fatalf("handler sent to many messages on the vertToMain channels")
			}
			// close would pannic because the listener isn't setup correctly -> skip it
			// if err := vert.Close(); err != nil {
			// 	test.Fatalf("failed to close vertical api: %v", err)
			// }
		})
	}
}

func TestWriteToConnection(test *testing.T) {
	test.Parallel()
	test.Run("notification", func(test *testing.T) {
		test.Parallel()
		var testLog *slog.Logger = slogt.New(test)
		cVert, cTest := net.Pipe()
		vertToMainChans := VertToMainChans{
			Register:   make(chan VertToMainRegister, 1),
			Anounce:    make(chan VertToMainAnnounce, 1),
			Validation: make(chan VertToMainValidation, 1),
		}
		vert := NewVerticalApi(testLog, vertToMainChans)

		mainToVert := make(chan MainToVertNotification, 1)
		go vert.writeToConnection(cVert, mainToVert)

		mainToVert <- MainToVertNotification{
			Data: vertTypes.GossipNotification{
				MessageHeader: vertTypes.MessageHeader{
					Size: 8 + 2,
					Type: vertTypes.GossipNotificationType,
				},
				MessageId: 1337,
				DataType:  42,
				Data:      []byte{0x50, 0x20},
			},
		}
		bufReal := []byte{0x0, 0x0a, 0x01, 0xf6, 0x05, 0x39, 0x0, 0x2a, 0x50, 0x20}

		// allow the message arrive within 5 seconds to avoid hanging up the test
		// TODO are 5 seconds too long? (makes these tests a bit slow)
		if err := cTest.SetReadDeadline(time.Now().Add(5*time.Second)); err != nil {
			test.Fatalf("Setting readDeadline failed: %v", err)
		}
		// receive the message sent on the network
		buf := make([]byte, len(bufReal))
		if n, err := io.ReadFull(cTest, buf); err != nil {
			test.Fatalf("Error reading from network. %v", err)
		} else if n != len(buf) {
			test.Fatalf("read too few bytes from the line. was: %d should: %d", n, len(buf))
		}
		if !reflect.DeepEqual(buf, bufReal) {
			test.Fatalf("Sent buffer is wrong. was: %v should: %v", buf, bufReal)
		}

		if err := cTest.SetReadDeadline(time.Now().Add(5*time.Second)); err != nil {
			test.Fatalf("Setting readDeadline failed: %v", err)
		}
		buf = make([]byte, 1)
		if n,err := cTest.Read(buf); err == nil || !errors.Is(err, os.ErrDeadlineExceeded) || n != 0 {
			test.Fatalf("There shouldn't be any data left on the socket: %v", err)
		}
	})
}

// mostly a combined version of the other two tests which also tests the tcp
// server and has a more black-box approach
func TestVerticalApi(test *testing.T) {
	var testLog *slog.Logger = slogt.New(test)
	vertToMainChans := VertToMainChans{
		Register:   make(chan VertToMainRegister, 1),
		Anounce:    make(chan VertToMainAnnounce, 1),
		Validation: make(chan VertToMainValidation, 1),
	}
	vert := NewVerticalApi(testLog, vertToMainChans)
	if err := vert.Listen("0.0.0.0:13377"); err != nil {
		test.Fatalf("Error starting server: %v", err)
	}

	cTest, err := net.Dial("tcp", "localhost:13377")
	if err != nil {
		test.Fatalf("Error connecting to server: %v", err)
	}

	t := tester{
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
	}

	if n, err := cTest.Write(t.buf); err != nil {
		test.Fatalf("failed sending: %v %+v", err, t)
	} else if n != len(t.buf) {
		test.Fatalf("buffer wasn't sent completely (might need to adjust the test) %v", t)
	}

	time.Sleep(5 * time.Second)

	var reg VertToMainRegister
	select {
	case reg = <-vertToMainChans.Register:
		t, ok := t.msg.(*vertTypes.GossipNotify)
		if !ok {
			test.Fatalf("handler sent to wrong channel")
		}
		if !reflect.DeepEqual(reg.Data, *t) {
			test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", reg.Data, *t)
		}
	default:
		test.Fatalf("handler didn't pass a message to the right vertToMain channel")
	}
	if len(vertToMainChans.Anounce) != 0 || len(vertToMainChans.Register) != 0 || len(vertToMainChans.Validation) != 0 {
		test.Fatalf("handler sent to many messages on the vertToMain channels %d %d %d", len(vertToMainChans.Anounce), len(vertToMainChans.Register), len(vertToMainChans.Validation))
	}

	mainToVert := reg.Module.MainToVert

	mainToVert <- MainToVertNotification{
		Data: vertTypes.GossipNotification{
			MessageHeader: vertTypes.MessageHeader{
				Size: 8 + 2,
				Type: vertTypes.GossipNotificationType,
			},
			MessageId: 1337,
			DataType:  42,
			Data:      []byte{0x50, 0x20},
		},
	}
	bufReal := []byte{0x0, 0x0a, 0x01, 0xf6, 0x05, 0x39, 0x0, 0x2a, 0x50, 0x20}

	// allow the message arrive within 5 seconds to avoid hanging up the test
	// TODO are 5 seconds too long? (makes these tests a bit slow)
	if err := cTest.SetReadDeadline(time.Now().Add(5*time.Second)); err != nil {
		test.Fatalf("Setting readDeadline failed: %v", err)
	}
	// receive the message sent on the network
	buf := make([]byte, len(bufReal))
	if n, err := io.ReadFull(cTest, buf); err != nil {
		test.Fatalf("Error reading from network. %v", err)
	} else if n != len(buf) {
		test.Fatalf("read too few bytes from the line. was: %d should: %d", n, len(buf))
	}
	if !reflect.DeepEqual(buf, bufReal) {
		test.Fatalf("Sent buffer is wrong. was: %v should: %v", buf, bufReal)
	}

	if err := cTest.SetReadDeadline(time.Now().Add(5*time.Second)); err != nil {
		test.Fatalf("Setting readDeadline failed: %v", err)
	}
	buf = make([]byte, 1)
	if n,err := cTest.Read(buf); err == nil || !errors.Is(err, os.ErrDeadlineExceeded) || n != 0 {
		test.Fatalf("There shouldn't be any data left on the socket: %v", err)
	}

	defer func() {
		if err := vert.Close(); err != nil {
			test.Fatalf("Failed to close server: %v", err)
		}
	}()
}
