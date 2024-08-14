# Security
- proof of work when connecting (2)
- periodic light proof of work (implemented in the (root) gossip strategy)

## PoW
- collect requests from neighbors -> do PoW on sequence of nonces -> respond
with sequence of nonces and PoW
- optimization: Peers can also ask for PoW challenge to include in the current
round
- DDOS protection: store when last PoW request was received from that peer
- Don't store state -> send (nonce, enc(nonce, ip, ...)) 

# Gossip Strategy
- [x] respect TTL
- send to conf param neighbors (instead of just one)
- think about pull maybe in the future

# Testing (py)
- test param to log with json -> add log level below debug for this
- create larger network, one node announces some data
  - Metrics:
    - #sent messages
    - who received it (distance, how many should received it, how many actually received it)
    - time axis

# Report
- state machine for sent messages etc
