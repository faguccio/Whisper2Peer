using Go = import "/go.capnp";
@0x92e5a6c1fee9dd14;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct ConnPoW $Go.doc("Send the PoW for the initial PoW on the horizontalApi.") {
	nonce      @0 :UInt64 $Go.doc("nonce which solves the PoW");
	cookie     @1 :Data   $Go.doc("encrypted data used at the responder to validate the PoW");
}
