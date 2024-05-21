# Project Ideas

## Project Layout

- Vertical API
    - Spawn a Thread per Connection
    - Thread is sync with MAIN using two channel (incoming, outcoming) messages

### Main
- 3 threads, list of (connection, types pair)
- Register types (GOSSIP NOTIFY)
- GOSSIP NOTIFY
- Relay messages (horizantal)

Is the entryPoint of GOSSIP MODULE

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


## Distribute Workload

