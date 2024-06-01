using Go = import "/go.capnp";
@0xd06424cd5634d6a3;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct Message {
	body :union {
		push @0 :import "push.capnp".PushMsg;
		empty @1 :Void;
	}
}
