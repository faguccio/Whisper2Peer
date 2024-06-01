using Go = import "/go.capnp";
@0xe8582155eeff22bb;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct PushMsg {
	ttl  @0 :UInt16;
	gossipType  @1 :UInt16;
	messageID  @2 :UInt16;
	payload  @3 :Data;
}
