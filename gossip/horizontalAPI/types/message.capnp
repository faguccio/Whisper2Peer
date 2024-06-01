using Go = import "/go.capnp";
@0xd06424cd5634d6a3;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct Message $Go.doc("Represents a generic message on the horizontalApi. Messagetype and length is done by capnproto somehow. empty might be removed in the future (union with only one member is not supported)") {
	body :union {
		push  @0 :import "push.capnp".PushMsg $Go.doc("message is a [PushMsg] message");
		empty @1 :Void                        $Go.doc("placeholder since union with only one member is not possible");
	}
}
