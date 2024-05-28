package main

import (
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"
	"sync"
)

// This struct is a dummy channel for specifying MainToVert channels where I can instruct the verticalapi module to send a Gossip Notification message

type notifyMap struct {
	data map[vertTypes.GossipType][]*verticalapi.RegisteredModule
	mu   sync.Mutex
}

func NewNotifyMap() *notifyMap {
	return &notifyMap{
		data: make(map[vertTypes.GossipType]([]*verticalapi.RegisteredModule)),
	}
}

func (nm *notifyMap) Load(gossip_type vertTypes.GossipType) []*verticalapi.RegisteredModule {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	res := nm.data[gossip_type]
	// if res == nil {
	// 	return []*verticalapi.RegisteredModule{}
	// }
	return res
}

func (nm *notifyMap) AddChannelToType(gossip_type vertTypes.GossipType, new_channel *verticalapi.RegisteredModule) {
	current_channels := nm.data[gossip_type]
	new_channels := append(current_channels, new_channel)
	nm.data[gossip_type] = new_channels
}
