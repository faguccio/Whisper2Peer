using Go = import "/go.capnp";
@0xb8aa23c3abf7cbd0;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct PowReq $Go.doc("Requesting a challenge for the periodic PoW on the horizontalApi.") {
	# no data needed when simply asking for a challenge
}
