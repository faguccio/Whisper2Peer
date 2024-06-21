package main

import (
	"errors"
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
func (nm *notifyMap) AddChannelToType(gossip_type common.GossipType, new_channel *common.Conn[common.RegisteredModule]) error {
	nm.Lock()
	defer nm.Unlock()

	{
		reg := nm.data[gossip_type]

		for _, conn := range reg {
			if conn.Id == new_channel.Id {
				return errors.New("Tried to register connection multiple times on type")
			}
		}
	}

	current_channels := nm.data[gossip_type]
	current_channels = append(current_channels, new_channel)
	nm.data[gossip_type] = current_channels
	return nil
}

func (nm *notifyMap) RemoveChannel(unreg common.ConnectionId) {
	nm.Lock()
	defer nm.Unlock()
	for k, l := range nm.data {
		for i, j := range l {
			if j.Id == unreg {
				l[i] = l[len(l)-1]
				nm.data[k] = l[:len(l)-1]
				break
			}
		}
	}
}
