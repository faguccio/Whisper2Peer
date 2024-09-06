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
