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

package main

import (
	gossip "gossip/main"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// init
	m := gossip.NewMain()

	// run
	initFin := make(chan error, 1)
	go m.Run(initFin)
	err := <-initFin
	if err != nil {
		panic(err)
	}

	// teardown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	m.Close()
}
