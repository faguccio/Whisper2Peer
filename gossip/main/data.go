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

package gossip

import (
	"errors"
	"gossip/common"
	"sync"
)

// This struct is a dummy channel for specifying MainToVert channels where I
// can instruct the verticalapi module to send a Gossip Notification message
//
// NewNotifyMap should be used to instanciate this.
type notifyMap struct {
	data map[common.GossipType][]*common.Conn[common.RegisteredModule]
	sync.RWMutex
}

// Use this function to instanciate the notifyMap
func NewNotifyMap() *notifyMap {
	return &notifyMap{
		data: make(map[common.GossipType]([]*common.Conn[common.RegisteredModule])),
	}
}

// getter for the nnotifyMap
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
				return errors.New("tried to register connection multiple times on type")
			}
		}
	}

	current_channels := nm.data[gossip_type]
	current_channels = append(current_channels, new_channel)
	nm.data[gossip_type] = current_channels
	return nil
}

// remove the connection with id == unreg from the notifyMap
//
// returns a pointer to the removed connection (or nil if no connection with id unreg was found)
func (nm *notifyMap) RemoveChannel(unreg common.ConnectionId) *common.Conn[common.RegisteredModule] {
	var ret *common.Conn[common.RegisteredModule]
	nm.Lock()
	defer nm.Unlock()
	for k, l := range nm.data {
		for i, j := range l {
			if j.Id == unreg {
				l[i] = l[len(l)-1]
				nm.data[k] = l[:len(l)-1]
				ret = j
				break
			}
		}
	}
	return ret
}
