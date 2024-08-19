# Proof of Work 

## Nonce for the PoW

When a peer A is asking for a proof of work, it needs to maintain state (I.E. about the challenge). Instead peer A will have a private symmetric key and it will send this state together with the challenge. Later A will be able to access such state by decrypting the corresponding part.

## Establishing a connection

Peer A is trying to connect to peer B. In the gossip strategy, before appending the connection to the open connection, the following steps are performed:

1. Peer A send a message `CONN_INIT` which can either contain the PoW or a special code asking for a the challenge
2. Peer B process the message. If it was a valid PoW, A is a new peer of B and B will send `CONN_OK` message. Otherwise, `B` will send a message `CONN_CHALL` which contains the challenge for the PoW.
3. If no `CONN_OK` was sent, connection is rejected

## Continuously asking for PoW

PoWs need to be asked continuously to avoid a sequential use of resources of an attacker. A global timer is used to synchronize challenges distribution. For this PoWs, there is no need to avoid maintaining a state, since the number of open connections is fixed. A peer A should perform the following steps:

1. A send a `KEEP_CHALL` message to all of its peers. 
2. A receive a `KEEP_CHALL` from (all) its peers. A keeps listening for a certain time window.
3. A compute one proof of work including all the `KEEP_CHALL` messages
4. A sends to all of its peers the PoW as a `KEEP_POW` message
5. A receive from all its peers a `KEEP_POW` message
6. If no or invalid `KEEP_POW` message is received, the connection is teared down

## Messages

### `CONN_INIT`

- `ChallReq`: if true, this is a challenge request
- `Nonce`: this is the string which is modified 
- `Cookie`: Enc(`Chall`, IP of both, timestamp)

### `CONN_CHALL`

- `Chall`: random integer
- `Cookie`: Enc(`Chall`, IP of both, timestamp)

### `CONN_OK`

- Empty message?

### `KEEP_CHALL`

- `Chall`: random integer
- `IP`: both IPs of peers

### `KEEP_POW`

- `Nonce`
- List of `KEEP_CHALL`