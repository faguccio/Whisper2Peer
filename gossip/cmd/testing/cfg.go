package main

import (
	"encoding/json"
	"os"
)

type Args struct {
	Fn          string `arg:"positional,required"`
	Degree      uint   `arg:"-d,--degree" default:"30" help:"Gossip parameter degree: Number of peers the current peer has to exchange information with"`
	Cache_size  uint   `arg:"-c,--cache" default:"50" help:"Gossip parameter cache_size: Maximum number of data items to be held as part of the peer’s knowledge base. Older items will be removed to ensure space for newer items if the peer’s knowledge base exceeds this limit"`
	GossipTimer uint   `arg:"-t,--gtimer" default:"1" help:"How often the gossip strategy should perform a strategy cycle, if applicable" default:"1"`
}

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

func readGraph(fn string) (graph,error) {
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

