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
	}
}
