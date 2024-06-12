package main

import (
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"
	"sync"
)

// This struct is a dummy channel for specifying MainToVert channels where I can instruct the verticalapi module to send a Gossip Notification message

type notifyMap struct {
	data       map[vertTypes.GossipType][]*verticalapi.RegisteredModule
	type_list  []vertTypes.GossipType
	cache_size int
	sync.RWMutex
}

func NewNotifyMap(cache_size int) *notifyMap {
	return &notifyMap{
		data:       make(map[vertTypes.GossipType]([]*verticalapi.RegisteredModule)),
		cache_size: cache_size,
	}
}

// Load function will register a new Gossip Type. Each type is a key and the value is the list of Channels
// which are listening for Gossip Notification of such type.
//
// If the number of registered types excede the cache size, the first registered type will be deleted
func (nm *notifyMap) Load(gossip_type vertTypes.GossipType) []*verticalapi.RegisteredModule {
	nm.RLock()
	defer nm.RUnlock()
	res := nm.data[gossip_type]

	return res
}

func (nm *notifyMap) AddChannelToType(gossip_type vertTypes.GossipType, new_channel *verticalapi.RegisteredModule) {
	nm.Lock()
	defer nm.Unlock()
	current_channels := nm.data[gossip_type]
	current_channels = append(current_channels, new_channel)

	nm.type_list = append(nm.type_list, gossip_type)
	if len(nm.type_list) > nm.cache_size {
		var deleted_type vertTypes.GossipType
		deleted_type, nm.type_list = nm.type_list[0], nm.type_list[1:]
		delete(nm.data, deleted_type)
		// TODO: DELETE ALL VERTICAL API CHANNELS
	}

	nm.data[gossip_type] = current_channels
}
