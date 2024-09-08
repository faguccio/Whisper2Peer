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

// Package packetcounter counts the amount of packets sent in a certain window
// of time.
package packetcounter

import (
	"sync"
	"time"
)

// counter represents the packetcounter
//
// You should be using [NewCounter] to instanciate new instances of the
// packetcounter
type Counter struct {
	t   time.Time
	cnt uint
	// gets called every time t is updated
	do func(t time.Time, cnt uint)
	// duration of the "buckets" to form
	granularity time.Duration
	mutex       sync.Mutex
}

// constructor for the packetcounter.
//
// `do` is the function called after the window of time has been exceeded. It
// will be provided with the start-time of the window and the amount of packets
// counted in that window of time.
//
// `granularity` is the duration of one window
func NewCounter(do func(time.Time, uint), granularity time.Duration) *Counter {
	return &Counter{
		t:           time.Time{},
		cnt:         0,
		do:          do,
		granularity: granularity,
	}
}

// count one packet at time.Now()
func (counter *Counter) Add(i uint) {
	counter.mutex.Lock()
	defer counter.mutex.Unlock()

	now := time.Now().Truncate(counter.granularity)
	if now == counter.t {
		counter.cnt += i
		return
	}

	if !counter.t.IsZero() {
		counter.do(counter.t, counter.cnt)
	}

	counter.cnt = i
	counter.t = now
}

// some packets might be counted but `do` was not called in the end. This
// function does the call to `do` if needed.
func (counter *Counter) Finalize() {
	counter.mutex.Lock()
	defer counter.mutex.Unlock()

	if !counter.t.IsZero() {
		counter.do(counter.t, counter.cnt)
	}
}
