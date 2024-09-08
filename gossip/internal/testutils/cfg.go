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
	"encoding/json"
	"os"
)

// Represents a node in the graph
type node struct {
	// use pointers for a quick and dirty optional
	Degree      *uint
	Cache_size  *uint
	GossipTimer *uint
}

// do custom unmarshalling to allow node to also be a simple integer (use
// default values for config in that case)
func (n node) UnmarshalJSON(data []byte) error {
	// try to unmarshal as integer
	var i uint
	err := json.Unmarshal(data, &i)
	if err == nil {
		n = node{}
		return nil
	}
	// unmarshal as int failed -> try unmarshal as node struct
	return json.Unmarshal(data, &n)
}

// Represents the complete Graph
type Graph struct {
	Nodes []node `json:"nodes"`
	// edges are represented as list of tuples (also modelled as list)
	Edges [][]uint `json:"edges"`
}

// read a graph from a json file
func NewGraphFromJSON(fn string) (Graph, error) {
	var g Graph
	f, err := os.Open(fn)
	if err != nil {
		return g, err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	err = d.Decode(&g)
	if err != nil {
		return g, err
	}

	return g, nil
}

// struct only used for bookkeeping when calculating the distances (with BFS)
type todo_bookkeeping struct {
	node uint
	dist uint
}

// use BFS to calculate all distances to the start node
//
// Returns a mapping from node(idx) to distance
func (g *Graph) CalcDistances(start uint) map[uint]uint {
	ret := make(map[uint]uint)

	edges := make(map[uint][]uint)
	for _, k := range g.Edges {
		edges[k[0]] = append(edges[k[0]], k[1])
		edges[k[1]] = append(edges[k[1]], k[0])
	}

	todo := []todo_bookkeeping{{start, 0}}
	for len(todo) > 0 {
		// obtain a new element for processing
		t := todo[0]
		todo = todo[1:]
		ret[t.node] = t.dist
		if ns, ok := edges[t.node]; !ok {
			continue
		} else {
			// go over neighbors
			for _, n := range ns {
				if _, ok := ret[n]; ok {
					// neighbor already visited
					continue
				}
				todo = append(todo, todo_bookkeeping{n, t.dist + 1})
			}
		}
	}

	// for all unconnected nodes -> set distance to uint-max
	for i := range g.Nodes {
		i := uint(i)
		if _, ok := ret[i]; !ok {
			ret[i] = ^uint(0)
		}
	}

	return ret
}
