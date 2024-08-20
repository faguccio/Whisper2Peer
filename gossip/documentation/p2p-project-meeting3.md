# Security
- proof of work when connecting (2)
- periodic light proof of work (implemented in the (root) gossip strategy)
- proof of work for PUSH messages (global limitation)

## PoW
- collect requests from neighbors -> do PoW on sequence of nonces -> respond
<<<<<<< Updated upstream
with sequence of nonces and PoW
  - optimization: Peers can also ask for PoW challenge to include in the current round
=======
with sequence of nonces and PoW (don't penalize node with lots of neighbors)
- optimization: Peers can also ask for PoW challenge to include in the current
round
>>>>>>> Stashed changes
- DDOS protection: store when last PoW request was received from that peer
- Don't store state -> send (nonce, enc(nonce, ip, ...)) 


## PUSH Messages

Push Message: add field for PoW nonce. The content to be hashed should be something like:

- variable part + (ID of messages?) + ID/IP of recipient

So that every node can verify received PUSH messages, without sending a challenge (IP will make it that PoW needs to be recomputed for each node). Replay attack are mitigated as messages received stay in cache as long as possible.



# Gossip Strategy
- [x] respect TTL
- [x] send to conf param neighbors (instead of just one)
- Implement PULL messages 

# Testing (py)
- test param to log with json -> add log level below debug for this
- create larger network, one node announces some data
  - Metrics:
    - #sent messages
    - who received it (distance, how many should received it, how many actually received it)
    - time axis

# Project "Deployment"

- [ ] Add Dockerfile
- [ ] Read from `.ini`

# Report
- state machine for sent messages etc
