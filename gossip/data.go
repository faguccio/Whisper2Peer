package main

import (
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"
	"sync"
)

// This struct is a dummy channel for specifying MainToVert channels where I can instruct the verticalapi module to send a Gossip Notification message

type notifyMap struct {
	data sync.Map
}

func NewNotifyMap() *notifyMap {
	return &notifyMap{
		data: sync.Map{},
	}
}

func (nm *notifyMap) Load(gossip_type vertTypes.GossipType) []*verticalapi.RegisteredModule {
	res, _ := nm.data.Load(gossip_type)
	if res == nil {
		return []*verticalapi.RegisteredModule{}
	}
	result, ok := res.([]*verticalapi.RegisteredModule)
	if !ok {
		panic("Tried to store in the NotifyMap a value which is not of type GossipType")
	}
	return result
}

func (nm *notifyMap) AddChannelToType(gossip_type vertTypes.GossipType, new_channel *verticalapi.RegisteredModule) {
	current_channels := nm.Load(gossip_type)
	new_channels := append(current_channels, new_channel)
	nm.data.Store(gossip_type, new_channels)
}
