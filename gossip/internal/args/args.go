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

// Package args is used for specifying the arguments (internally) used by the application.
package args

// Represents the arguments used actually/internally by the application

// You can use [NewFromDefaults] to obtain this struct with default values set
type Args struct {
	// Gossip parameter degree: Number of peers the current peer has to
	// exchange information with
	Degree uint
	// Gossip parameter cache_size: Maximum number of data items to be held as
	// part of the peer’s knowledge base. Older items will be removed to ensure
	// space for newer items if the peer’s knowledge base exceeds this limit
	Cache_size uint
	// How often the gossip strategy should perform a strategy cycle, if
	// applicable
	GossipTimer uint
	// Address to listen for incoming peer connections, ip:port
	Hz_addr string
	// Address to listen for incoming peer connections, ip:port
	Vert_addr string
	// List of horizontal peers to connect to, [ip]:port
	Peer_addrs []string
	// Strategy string
}

// Returns a new [Args] struct with sane default values
func NewFromDefaults() Args {
	return Args{
		Degree:      30,
		Cache_size:  50,
		GossipTimer: 1,
		Hz_addr:     "127.0.0.1:6001",
		Vert_addr:   "127.0.0.1:7001",
		Peer_addrs:  nil,
	}
}
