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

package testutils

import (
	"context"
	"encoding/json"
	"gossip/common"
	"io"
	"log/slog"
	"time"

	testlog "gossip/internal/testLog"
)

func LogInit(w io.Writer, id common.ConnectionId) *slog.Logger {
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

type Event struct {
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

func FilterLog(ctx context.Context, c chan<- Event, r io.Reader) {
	d := json.NewDecoder(r)
	for d.More() {
		var e Event
		err := d.Decode(&e)
		if err != nil {
			panic(err)
		}
		if e.Level != int(common.LevelTest) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		default:
			c <- e
		}
	}
}
