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
@0x979d950d9da073e2;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct PowChall $Go.doc("Respond with a new challenge for the periodic PoW on the horizontalApi.") {
	cookie    @0 :Data   $Go.doc("encrypted data used at the responder to validate the PoW, also serves as challenge");
}
