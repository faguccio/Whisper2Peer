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
@0xd06424cd5634d6a3;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct Message $Go.doc("Represents a generic message on the horizontalApi. Messagetype and length is done by capnproto somehow.") {
	body :union {
		push       @0 :import "push.capnp".PushMsg             $Go.doc("message is a [PushMsg] message");
		connChall  @1 :import "conn_challenge.capnp".ConnChall $Go.doc("message is a [ConnChall] message used int the initial PoW");
		connPoW    @2 :import "conn_pow.capnp".ConnPoW         $Go.doc("message is a [ConnPoW] message used int the initial PoW");
		connReq    @3 :import "conn_request.capnp".ConnReq     $Go.doc("message is a [ConnReq] message used int the initial PoW");
		powChall   @4 :import "pow_challenge.capnp".PowChall   $Go.doc("message is a [PowChall] message used int the periodic PoW");
		powPoW     @5 :import "pow_pow.capnp".PowPoW           $Go.doc("message is a [PowPoW] message used int the periodic PoW");
		powReq     @6 :import "pow_request.capnp".PowReq       $Go.doc("message is a [PowReq] message used int the periodic PoW");
	}
}
