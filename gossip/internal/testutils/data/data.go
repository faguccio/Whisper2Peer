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

// Package data holds struct definitions for the data which can be obtained by
// benchmarking. Usually the types should be marshallable via a css library or
// a `Write*` should be defined to write the data to file.
package data

import (
	"fmt"
	"os"
)

type ReachedWhenAll map[uint]ReachedWhen
type ReachedWhen struct {
	TimeUnixSec float64
	TimePercent float64
}

// generate a css style sheet to draw colored graph
func (r *ReachedWhenAll) WriteCss(fn string) error {
	f, err := os.Create(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for nodeIdx, reached := range *r {
		if reached.TimePercent <= 50 {
			fmt.Fprintf(f, `._%d>ellipse {
    fill: color-mix(in srgb, yellow %d%%, green);
}
`, nodeIdx, uint(reached.TimePercent*2))
		} else if reached.TimePercent <= 100 {
			fmt.Fprintf(f, `._%d>ellipse {
    fill: color-mix(in srgb, red %d%%, yellow);
}
`, nodeIdx, uint(reached.TimePercent*2-100))
		}
	}

	return nil
}

type ReachedDistCntAll []ReachedDistCnt
type ReachedDistCnt struct {
	TimeUnixSec            float64
	Distance               uint
	CntReachedSameDistance uint
}

type CntDistancesAll []CntDistances
type CntDistances struct {
	Distance uint
	Cnt      uint
}

type SentPacketsCntAll []SentPacketsCnt
type SentPacketsCnt struct {
	TimeUnixSec float64
	Cnt         uint
}
