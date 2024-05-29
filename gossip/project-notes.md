# Project Notes

- Vertical API
    - Spawn a Thread per Connection
    - Thread is sync with MAIN using two channel (incoming, outcoming) messages

## Main
Is the entryPoint of GOSSIP MODULE. The main functionalities of main are:

1. Relaying messages that come through either a Gossip Announce message or the Horizontal API
    - The latter will reach main just for validation, or we could share the data structure with horizontalapi
2. Handle the Gossip Notify Messages by adding them to a data structure
    - [TODO] We need to decide how main to vert communication is done. It could be one single channel with identifiers in the message (verticalapi is doing the sorting) 

### Types Map

A map is used to store information about the Gossip Types we need to spread and send notification for. It should e clarified if the types need to be cleared if all connection get closed. Also we might forget to remove connections from the map.

```
{
    "notify Type": "[channels to relay the message to]"
}
```

### Channels

Something that is getting pretty clear is that messages reaching main need to have a connection identifier. We could either generate one mainToVert (and horz) channel for each new connection (so it would be very fast for main to send Notification to the right channel).


### Horizontal API
 
- How to identify messages for PULL
    - Hash the message
    - Combination of (SENDER_ID, RANDOM)

- Web of Trust
    - Endorse messages


## Current commit changes



