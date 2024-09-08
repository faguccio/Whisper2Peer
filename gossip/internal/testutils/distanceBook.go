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
	"slices"
)

// some orga stuff to avoid calculating the distances too often
// this struct only contains entries which will/must not change during
// processing (therefore no dynamic distCnt included)
type distanceBook struct {
	// store if distanceBook was already initialized
	valid bool
	// store the startNode for which the distances were calculated
	startNode uint
	// actual generated data
	nodeToDist map[uint]uint
	distOrd    []uint
	distMaxCnt map[uint]uint
}

// setup working with distances with the given start node. If the distances for
// this start node are already calculated, don't re-calculate the distances
func (db *distanceBook) processingSetupForDistance(genDistances func(uint) map[uint]uint, startNode uint) map[uint]uint {
	distCnt := make(map[uint]uint)
	if db.valid && db.startNode == startNode {
		// init map with known distances
		for d := range db.distMaxCnt {
			distCnt[d] = 0
		}
		return distCnt
	}

	db.nodeToDist = genDistances(startNode)
	db.distMaxCnt = make(map[uint]uint)
	// collect (and count) known distances
	for _, v := range db.nodeToDist {
		if _, ok := distCnt[v]; !ok {
			distCnt[v] = 0
			db.distMaxCnt[v] = 0
			db.distOrd = append(db.distOrd, v)
		}
		db.distMaxCnt[v] += 1
	}
	slices.Sort(db.distOrd)

	db.valid = true
	db.startNode = startNode

	return distCnt
}
