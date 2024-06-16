package main

import (
	"gossip/common"
	"sync"
)

// This struct is a dummy channel for specifying MainToVert channels where I can instruct the verticalapi module to send a Gossip Notification message

type notifyMap struct {
	data map[common.GossipType][]*common.Conn[common.RegisteredModule]
	sync.RWMutex
}

func NewNotifyMap() *notifyMap {
	return &notifyMap{
		data: make(map[common.GossipType]([]*common.Conn[common.RegisteredModule])),
	}
}

func (nm *notifyMap) Load(gossip_type common.GossipType) []*common.Conn[common.RegisteredModule] {
	nm.RLock()
	defer nm.RUnlock()
	res := nm.data[gossip_type]

	return res
}

// AddChannelToType function will register a new Gossip Type. Each type is a key and the value is the list of Channels
// which are listening for Gossip Notification of such type.
//
// If the number of registered types excede the cache size, the first registered type will be deleted
func (nm *notifyMap) AddChannelToType(gossip_type common.GossipType, new_channel *common.Conn[common.RegisteredModule]) {
	// TODO can register multiple times with same type?
	nm.Lock()
	defer nm.Unlock()
	current_channels := nm.data[gossip_type]
	current_channels = append(current_channels, new_channel)
	nm.data[gossip_type] = current_channels
}

func (nm *notifyMap) RemoveChannel(unreg common.GossipUnRegister) {
	nm.Lock()
	defer nm.Unlock()
	for k, l := range nm.data {
		for i, j := range l {
			if j.Id == common.ConnectionId(unreg) {
				l[i] = l[len(l)-1]
				nm.data[k] = l[:len(l)-1]
				break
			}
		}
	}
}
