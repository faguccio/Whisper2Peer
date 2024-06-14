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
