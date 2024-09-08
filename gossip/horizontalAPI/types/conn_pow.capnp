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
@0x92e5a6c1fee9dd14;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct ConnPoW $Go.doc("Send the PoW for the initial PoW on the horizontalApi.") {
	nonce      @0 :UInt64 $Go.doc("nonce which solves the PoW");
	cookie     @1 :Data   $Go.doc("encrypted data used at the responder to validate the PoW");
}
