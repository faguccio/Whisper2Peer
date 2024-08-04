package main

import (
	"encoding/json"
	"os"
)

// quick and dirty to detect if value was set or not
type node struct {
	Degree      *uint
	Cache_size  *uint
	GossipTimer *uint
}

func (n node) UnmarshalJSON(data []byte) error {
	var i uint
	err := json.Unmarshal(data, &i)
	if err == nil {
		n = node{}
		return nil
	}
	return json.Unmarshal(data, &n)
}

type graph struct {
	Nodes []node   `json:"nodes"`
	Edges [][]uint `json:"edges"`
}

type todo_bookkeeping struct {
	node uint
	dist uint
}

func (g *graph) calcDistances(start uint) map[uint]uint {
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

func readGraph(fn string) (graph, error) {
	var g graph
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
