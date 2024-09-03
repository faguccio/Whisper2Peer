using Go = import "/go.capnp";
@0x979d950d9da073e2;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct PowChall $Go.doc("Respond with a new challenge for the periodic PoW on the horizontalApi.") {
	cookie    @0 :Data   $Go.doc("encrypted data used at the responder to validate the PoW, also serves as challenge");
}
