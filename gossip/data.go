package main

import (
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"
	"sync"
)

// This struct is a dummy channel for specifying MainToVert channels where I can instruct the verticalapi module to send a Gossip Notification message

type notifyMap struct {
	data map[vertTypes.GossipType][]*verticalapi.RegisteredModule
	sync.RWMutex
}

func NewNotifyMap() *notifyMap {
	return &notifyMap{
		data: make(map[vertTypes.GossipType]([]*verticalapi.RegisteredModule)),
	}
}

func (nm *notifyMap) Load(gossip_type vertTypes.GossipType) []*verticalapi.RegisteredModule {
	nm.RLock()
	defer nm.RUnlock()
	res := nm.data[gossip_type]

	return res
}

// AddChannelToType function will register a new Gossip Type. Each type is a key and the value is the list of Channels
// which are listening for Gossip Notification of such type.
//
// If the number of registered types excede the cache size, the first registered type will be deleted
func (nm *notifyMap) AddChannelToType(gossip_type vertTypes.GossipType, new_channel *verticalapi.RegisteredModule) {
	nm.Lock()
	defer nm.Unlock()
	current_channels := nm.data[gossip_type]
	current_channels = append(current_channels, new_channel)
	nm.data[gossip_type] = current_channels
}
