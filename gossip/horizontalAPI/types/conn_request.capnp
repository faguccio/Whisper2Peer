using Go = import "/go.capnp";
@0xa29b5ee0b8b34771;
$Go.package("types");
$Go.import("gossip/horizontalAPI/types");

struct ConnReq $Go.doc("Requesting a challenge for the initial PoW on the horizontalApi.") {
	# no data needed when simply asking for a challenge
}
