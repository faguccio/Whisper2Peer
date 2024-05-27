package main

import (
	vertTypes "gossip/verticalAPI/types"
	"slices"
	"sync"
)

// This struct is a dummy channel for specifying MainToVert channels where I can instruct the verticalapi module to send a Gossip Notification message
type NotificationChannel struct {
	NotificationChannel <-chan string
}

type notifyMap struct {
	data sync.Map
}

func NewNotifyMap() *notifyMap {
	return &notifyMap{
		data: sync.Map{},
	}
}

func (nm *notifyMap) Load(gossip_type vertTypes.GossipType) []NotificationChannel {
	res, _ := nm.data.Load(gossip_type)
	if res == nil {
		return []NotificationChannel{}
	}
	result, ok := res.([]NotificationChannel)
	if !ok {
		panic("Tried to store in the NotifyMap a value which is not of type GossipType")
	}
	return result
}

func (nm *notifyMap) AddChannelToType(gossip_type vertTypes.GossipType, new_channel NotificationChannel) {
	current_channels := nm.Load(gossip_type)
	new_channels := slices.Concat(current_channels, []NotificationChannel{new_channel})
	nm.data.Store(gossip_type, new_channels)
}
