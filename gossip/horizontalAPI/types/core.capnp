using Go = import "/go.capnp";
@0xb58493dca0c0699d;
$Go.package("horizontalAPI");
$Go.import("gossip/horizontalAPI");

enum HorizontalTypes {
	push @0;
}

struct PushMsg {
	horizontalType @0 :UInt16;
	ttl  @1 :UInt16;
	gossipType  @2 :UInt16;
	messageID  @3 :UInt16;
	payload  @4 :Data;
}
