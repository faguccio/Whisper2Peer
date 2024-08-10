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
	distOrd []uint
	distMaxCnt map[uint]uint
}

// setup working with distances with the given start node. If the distances for
// this start node are already calculated, don't re-calculate the distances
func (db *distanceBook) processingSetupForDistance(genDistances func(uint)map[uint]uint, startNode uint) (map[uint]uint) {
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
