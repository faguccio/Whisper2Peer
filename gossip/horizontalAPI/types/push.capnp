# gossip
# Copyright (C) 2024 Fabio Gaiba and Lukas Heindl
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

using Go = import "/go.capnp";
@0xe8582155eeff22bb;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct PushMsg $Go.doc("Represents a push message on the horizontalApi.") {
	ttl         @0 :UInt8 $Go.doc("time-to-live: how many further hops should the message be propagated");
	# gossip type might become an enum at some point
	gossipType  @1 :UInt16 $Go.doc("type of the payload");
	messageID   @2 :UInt16 $Go.doc("identification of the message");
	payload     @3 :Data   $Go.doc("arbitrary payload");
}
