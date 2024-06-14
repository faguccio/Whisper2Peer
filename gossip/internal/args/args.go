package args

type Args struct {
	Degree     int      `arg:"-d,--degree" default:"30" help:"Gossip parameter degree: Number of peers the current peer has to exchange information with"`
	Cache_size int      `arg:"-c,--cache" default:"50" help:"Gossip parameter cache_size: Maximum number of data items to be held as part of the peer’s knowledge base. Older items will be removed to ensure space for newer items if the peer’s knowledge base exceeds this limit"`
	Hz_addr    string   `arg:"-h,--haddr" default:"127.0.0.1:6001" help:"Address to listen for incoming peer connections, ip:port"`
	Vert_addr  string   `arg:"-v,--vaddr" default:"127.0.0.1:7001" help:"Address to listen for incoming peer connections, ip:port"`
	Peer_addrs []string `arg:"positional" help:"List of horizontal peers to connect to, [ip]:port"`
	// Strategy string ``
}
