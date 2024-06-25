package main

import (
	"encoding/json"
	"fmt"
	"gossip/common"
	"io"
	"log/slog"
	"time"
)


func logInit(w io.Writer, id common.ConnectionId) *slog.Logger {
	log := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:     common.LevelTest,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				a.Value = slog.IntValue(int(level))
			}
			return a
		},
		},
	))
	return log.With("id", id)
}

type event struct {
	Time *time.Time
	Level int
	Msg string
	Id common.ConnectionId
}

func filterLog(c chan<- event, r io.Reader) {
	d := json.NewDecoder(r)
	for d.More() {
		var e event
		err := d.Decode(&e)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", e)
		if e.Level != -8 {
			continue
		}
		c <- e
	}
}
