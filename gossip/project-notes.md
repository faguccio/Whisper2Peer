# Project Notes

- Vertical API
    - Spawn a Thread per Connection
    - Thread is sync with MAIN using two channel (incoming, outcoming) messages

### Main
Is the entryPoint of GOSSIP MODULE. It has 3 main threads:

1. Relaying messages that come through either a Gossip Announce message or the Horizontal API

- 3 threads, list of (connection, types pair)
- Register types (GOSSIP NOTIFY)
- GOSSIP NOTIFY
- Relay messages (horizantal)



```go
struct {
    vertToMain channel
    mainToVert channel (x3)
}
```

### Horizontal API
 
- How to identify messages for PULL
    - Hash the message
    - Combination of (SENDER_ID, RANDOM)

- Web of Trust
    - Endorse messages


## Current commit changes

Created the horizontalapi package with a dummy interface.

