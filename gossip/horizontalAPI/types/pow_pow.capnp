using Go = import "/go.capnp";
@0xd6efd2f95fb94bec;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct PowPoW $Go.doc("Send the PoW for the periodic PoW on the horizontalApi.") {
	nonce      @0 :UInt64 $Go.doc("nonce which solves the PoW");
	cookie     @1 :Data   $Go.doc("encrypted data used at the responder to validate the PoW");
}
