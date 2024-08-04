package main

import (
	"encoding/json"
	"gossip/common"
	"io"
	"log/slog"
	"time"

	"gossip/internal/testLog"
)

func logInit(w io.Writer, id common.ConnectionId) *slog.Logger {
	log := slog.New(testlog.NewTestHandler(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: common.LevelTest,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				a.Value = slog.IntValue(int(level))
			}
			return a
		},
	},
	), common.LevelTest))
	return log.With("id", id)
}

type event struct {
	// logging items
	Time  time.Time
	Level int
	Msg   string
	// artificially added items (fixed)
	Id common.ConnectionId
	// what packet was received
	MsgId   uint16
	MsgType common.GossipType
	// how many packets were sent
	Cnt        uint
	TimeBucket time.Time
}

func filterLog(c chan<- event, r io.Reader) {
	d := json.NewDecoder(r)
	for d.More() {
		var e event
		err := d.Decode(&e)
		if err != nil {
			panic(err)
		}
		if e.Level != int(common.LevelTest) {
			continue
		}
		c <- e
	}
}
