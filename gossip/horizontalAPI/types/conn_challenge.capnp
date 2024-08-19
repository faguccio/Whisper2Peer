using Go = import "/go.capnp";
@0x9f9988149b4d7a10;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct ConnChall $Go.doc("Respond with a new challenge for the initial PoW on the horizontalApi.") {
	challenge @0 :UInt64 $Go.doc("actual challenge for the PoW");
	cookie    @1 :Data   $Go.doc("encrypted data used at the responder to validate the PoW");
}
