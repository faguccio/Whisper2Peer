package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/lmittmann/tint"
	ini "gopkg.in/ini.v1"
)

var USE_PARALLEL int = 2
var LOG_OUTPUT bool = true
const CONFIG_FILE = "config.ini"
const WORKERS = 32

func sequentialProofOfWork(pred func(digest []byte) bool, e EnrollRegister) uint64 {
	h := sha256.New()
	h.Reset()
	digest := [sha256.Size]byte{}

	m := make([]byte, 0)
	m,_ = e.Marshal(m)
	m = m[binary.Size(e.Challenge):]

	for e.Nonce = 0; true; e.Nonce++ {
		h.Write(binary.BigEndian.AppendUint64(nil, e.Challenge))
		h.Write(m)
		if pred(h.Sum(digest[:0])) {
			return e.Nonce
		}
	}
	return 0
}

func parallelProofOfWork(pred func(digest []byte) bool, e EnrollRegister) uint64 {
	stream := make(chan uint64, 32)
	result := make(chan uint64, 1)
	for i := 0; i < WORKERS; i++ {
		go func(){
			h := sha256.New()
			h.Reset()
			digest := [sha256.Size]byte{}

			m := make([]byte, 0)
			m,_ = e.Marshal(m)
			m = m[binary.Size(e.Challenge):]

			loop:
			for e.Nonce = range stream {
				h.Write(binary.BigEndian.AppendUint64(nil, e.Challenge))
				h.Write(m)
				if pred(h.Sum(digest[:0])) {
					select {
					case result <- e.Nonce:
					default:
					}
					break loop
				}
			}
		}()
	}
	for i := uint64(0); true; i++ {
		select {
		case r := <- result:
			close(stream)
			return r
		default:
			stream <- i
		}
	}
	panic("invalid state")
}

func parallelProofOfWork2(pred func(digest []byte) bool, e EnrollRegister) uint64 {
	result := make(chan uint64)
	ctx,cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for i := 0; i < WORKERS; i++ {
		wg.Add(1)
		// c,_ := context.WithCancel(ctx)
		go func(ctx context.Context, id uint64){
			h := sha256.New()
			digest := [sha256.Size]byte{}
			h.Reset()

			m := make([]byte, 0)
			m,_ = e.Marshal(m)
			m = m[binary.Size(e.Challenge):]

			loop:
			for e.Nonce = id; true; e.Nonce+=uint64(WORKERS) {
				// print(e.Nonce, "\n")
				select {
				case <-ctx.Done():
					break loop
				default:
					h.Write(binary.BigEndian.AppendUint64(nil, e.Challenge))
					h.Write(m)
					if pred(h.Sum(digest[:0])) {
						select {
						case result <- e.Nonce:
						default:
						}
						break loop
					}
				}
			}
			wg.Done()
		}(ctx, uint64(i))
	}
	r := <- result
	cancel()
	wg.Wait()
	return r
}

func first24bits0(digest []byte) bool {
	return digest[0] == 0 && digest[1] == 0 && digest[2] == 0
}

type MessageHeader struct {
	Size uint16
	Type uint16
}

func (m *MessageHeader) Unmarshal(buf []byte) (int, error) {
	m.Size = binary.BigEndian.Uint16(buf)
	m.Type = binary.BigEndian.Uint16(buf[2:])
	return 32, nil
}
func (m *MessageHeader) Hash(h hash.Hash) hash.Hash {
	if h == nil {
		h = sha256.New()
	}
	h.Write(binary.BigEndian.AppendUint16(nil, m.Size))
	h.Write(binary.BigEndian.AppendUint16(nil, m.Type))
	return h
}
func (m *MessageHeader) Marshal(buf []byte) error {
	binary.BigEndian.PutUint16(buf, m.Size)
	binary.BigEndian.PutUint16(buf[2:], m.Type)
	return nil
}
func (m *MessageHeader) CalcSize() int {
	return binary.Size(m)
}

const EnrollInitType = 680

type EnrollInit struct {
	MessageHeader
	Challenge uint64
}

func (e *EnrollInit) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != EnrollInitType {
		return 0, errors.New("wrong type")
	}
	e.Challenge = binary.BigEndian.Uint64(buf[e.MessageHeader.CalcSize():])
	return 64, nil
}
func (e *EnrollInit) CalcSize() int {
	return binary.Size(e)
}

const EnrollRegisterType = 681

type EnrollRegister struct {
	MessageHeader `ini:"-"`
	Challenge     uint64 `ini:"-"`
	TeamNumber    uint16 `ini:"teamNumber"`
	ProjectChoice uint16 `ini:"projectChoice"`
	Nonce         uint64 `ini:"-"`
	Email         string `ini:"email"`
	Firstname     string `ini:"firstname"`
	Lastname      string `ini:"lastname"`
	GitlabHandle  string `ini:"gitlabHandle"`
}

func (e *EnrollRegister) Hash(h hash.Hash) hash.Hash {
	if h == nil {
		h = sha256.New()
	}
	h.Write(binary.BigEndian.AppendUint64(nil, e.Challenge))
	h.Write(binary.BigEndian.AppendUint16(nil, e.TeamNumber))
	h.Write(binary.BigEndian.AppendUint16(nil, e.ProjectChoice))
	h.Write(binary.BigEndian.AppendUint64(nil, e.Nonce))
	h.Write([]byte(e.Email))
	h.Write([]byte("\r\n"))
	h.Write([]byte(e.Firstname))
	h.Write([]byte("\r\n"))
	h.Write([]byte(e.Lastname))
	h.Write([]byte("\r\n"))
	h.Write([]byte(e.GitlabHandle))
	return h
}
func (e *EnrollRegister) Marshal(buf []byte) ([]byte, error) {
	if e.MessageHeader.Type != EnrollRegisterType {
		return nil, errors.New("wrong type")
	}
	buf = slices.Grow(buf, e.CalcSize()+3*2)
	buf = buf[:e.CalcSize()+3*2]
	if err := e.MessageHeader.Marshal(buf); err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint64(buf[e.MessageHeader.CalcSize():], e.Challenge)
	binary.BigEndian.PutUint16(buf[e.MessageHeader.CalcSize()+8:], e.TeamNumber)
	binary.BigEndian.PutUint16(buf[e.MessageHeader.CalcSize()+10:], e.ProjectChoice)
	binary.BigEndian.PutUint64(buf[e.MessageHeader.CalcSize()+12:], e.Nonce)
	copy(buf[e.MessageHeader.CalcSize()+20:], fmt.Sprintf("%s\r\n%s\r\n%sn%s", e.Email, e.Firstname, e.Lastname, e.GitlabHandle))
	return buf, nil
}
func (e *EnrollRegister) FillFromIni(fn string) error {
	cfg, err := ini.Load(fn)
	if err != nil {
		return err
	}
	if err = cfg.Section("registration").MapTo(e); err != nil {
		return err
	}
	e.MessageHeader.Type = EnrollRegisterType
	return nil
}
func (e *EnrollRegister) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.Challenge)
	s += binary.Size(e.TeamNumber)
	s += binary.Size(e.ProjectChoice)
	s += binary.Size(e.Nonce)
	s += len(e.Email)
	s += len(e.Firstname)
	s += len(e.Lastname)
	s += len(e.GitlabHandle)
	return s
}

const EnrollSuccessType = 682

type EnrollSuccess struct {
	MessageHeader
	Reserved   uint16
	TeamNumber uint16
}

func (e *EnrollSuccess) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != EnrollSuccessType {
		return 0, errors.New("wrong type")
	}
	e.Reserved = binary.BigEndian.Uint16(buf[e.MessageHeader.CalcSize():])
	e.TeamNumber = binary.BigEndian.Uint16(buf[e.MessageHeader.CalcSize()+2:])
	return 32, nil
}
func (e *EnrollSuccess) CalcSize() int {
	return binary.Size(e)
}

const EnrollFailureType = 683

type EnrollFailure struct {
	MessageHeader
	Reserved    uint16
	ErrorNumber uint16
	Description string
}

func (e *EnrollFailure) Unmarshal(buf []byte) (int, error) {
	if e.MessageHeader.Type != EnrollFailureType {
		return 0, errors.New("wrong type")
	}
	e.Reserved = binary.BigEndian.Uint16(buf[e.MessageHeader.CalcSize():])
	e.ErrorNumber = binary.BigEndian.Uint16(buf[e.MessageHeader.CalcSize()+2:])
	desc := make([]byte, int(e.MessageHeader.Size)-e.MessageHeader.CalcSize()-2-2)
	copy(desc, buf[e.MessageHeader.CalcSize()+4:])
	e.Description = string(desc)
	return 32 + len(e.Description), nil
}
func (e *EnrollFailure) CalcSize() int {
	s := e.MessageHeader.CalcSize()
	s += binary.Size(e.Reserved)
	s += binary.Size(e.ErrorNumber)
	s += len(e.Description)
	return s
}

const HOST = "p2psec.net.in.tum.de:13337"

func logInit() *slog.Logger {
	if LOG_OUTPUT {
		return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}))
	} else {
		return slog.New(tint.NewHandler(io.Discard, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}))
	}
}

func main() {
	var err error
	// set up logging
	slog.SetDefault(logInit())

	conn, err := net.Dial("tcp", HOST)
	if err != nil {
		panic(err)
	}

	sentPackets := 0

	loop:
	for {
		// parse the message header
		var msgHdr MessageHeader
		buffer := make([]byte, msgHdr.CalcSize())
		read_size, err := conn.Read(buffer)
		if err != nil {
			panic(err)
		}
		if read_size != len(buffer) {
			panic(errors.New("didn't read enough bytes"))
		}
		slog.Debug("incoming packet header", "buffer", buffer)

		parsed_size, err := msgHdr.Unmarshal(buffer)
		if err != nil {
			panic(err)
		}
		slog.Debug("unmarshalled message header", "messageHdr", msgHdr)

		// read body
		// grow capacity
		buffer = slices.Grow(buffer, int(msgHdr.Size))
		// grow length as well
		buffer = buffer[:int(msgHdr.Size)]
		tmp := read_size
		read_size, err = conn.Read(buffer[read_size:])
		if err != nil {
			panic(err)
		}
		if read_size != len(buffer[tmp:]) {
			panic(errors.New("didn't read enough bytes"))
		}
		slog.Debug("incoming packet", "buffer", buffer[tmp:])

		switch msgHdr.Type {

		case EnrollInitType:
			// parse the enrollInit message
			enrollInit := EnrollInit{
				MessageHeader: msgHdr,
			}
			parsed_size, err = enrollInit.Unmarshal(buffer)
			if err != nil {
				panic(err)
			}
			_ = parsed_size
			slog.Info("unmarshalled message", "enroll init", enrollInit)

			// build the register message
			var enrollReg EnrollRegister
			enrollReg.FillFromIni(CONFIG_FILE)
			// calculate size + add space for \r\n
			enrollReg.Size = uint16(enrollReg.CalcSize()) + 2*3
			enrollReg.Challenge = enrollInit.Challenge

			switch USE_PARALLEL {
			case 0:
				enrollReg.Nonce = sequentialProofOfWork(first24bits0, enrollReg)
				slog.Debug("register message", "struct", enrollReg, "hash", fmt.Sprintf("%32x", enrollReg.Hash(nil).Sum(nil)))
			case 1:
				enrollReg.Nonce = parallelProofOfWork(first24bits0, enrollReg)
				slog.Debug("register message", "struct", enrollReg, "hash", fmt.Sprintf("%32x", enrollReg.Hash(nil).Sum(nil)))
			case 2:
				enrollReg.Nonce = parallelProofOfWork2(first24bits0, enrollReg)
				slog.Debug("register message", "struct", enrollReg, "hash", fmt.Sprintf("%32x", enrollReg.Hash(nil).Sum(nil)))
			}

			// clear buffer (not strictly needed if we are careful with the length)
			buffer = make([]byte, 0)
			buffer, err = enrollReg.Marshal(buffer)
			if err != nil {
				panic(err)
			}
			slog.Debug("marshalled message", "enroll register", buffer)

			if sentPackets > 1 {
				panic(errors.New("client is expected to only send a packet once"))
			}

			cfg, err := ini.Load(CONFIG_FILE)
			if err != nil {
				panic(err)
			}
			if cfg.Section("").Key("registered").MustBool(true) {
				slog.Info("you're already registered -> don't actually send the message. If you still want to register add 'registered=false' to your config")
				break loop
			}

			_, err = conn.Write(buffer)
			sentPackets++
			if err != nil {
				panic(err)
			}

		case EnrollSuccessType:
			// parse the enrollSuccess message
			enrollSuccess := EnrollSuccess{
				MessageHeader: msgHdr,
			}
			parsed_size, err = enrollSuccess.Unmarshal(buffer)
			if err != nil {
				panic(err)
			}
			_ = parsed_size
			slog.Info("unmarshalled message", "enroll success", enrollSuccess)
			break loop

		case EnrollFailureType:
			// parse the enrollFailure message
			enrollFailure := EnrollFailure{
				MessageHeader: msgHdr,
			}
			parsed_size, err = enrollFailure.Unmarshal(buffer)
			if err != nil {
				panic(err)
			}
			_ = parsed_size
			slog.Warn("unmarshalled message", "enroll failure", enrollFailure)
			break loop

		default:
			slog.Error("unknown packet type", "type", msgHdr.Type)
			panic(errors.New("unknown packet type"))
		}
	}
	defer conn.Close()
}
