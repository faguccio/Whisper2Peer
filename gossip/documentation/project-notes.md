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


## Horizontal API

The horizontalApi can both establish connections on its own (AddNeighbors) and accept incoming connections (Listen). It uses capnproto to read on/write to the connection.
A kinda typical usage of the module is given in the TestHorizontalApi test (creating two horizontalAPIs, one connects to the other one and one sends a message on the connection).
Note that the coverage is 75% mainly because some error branches are missing (it's hard to simulate some networking error reliably).

### Future

- How to identify messages for PULL
    - Hash the message
    - Combination of (SENDER_ID, RANDOM)

- Web of Trust
    - Endorse messages


## Testing

- Log with `JsonHandler` when a debug or testing flag is passed as an argument

## To resolve

Do we want to set the source address as well for `net.Dial`? (`usenet.Dialer{}.Dial`) If so where to specify the desired address?


## Notes for the report

About capnproto
There is the comment //go:generate capnp compile -I $HOME/programme/go-capnp/std -ogo:./ types/message.capnp types/push.capnp which causes go generate run the capnp command which does the code generation (see the target in the Makefile). To avoid the necessity of everyone having to install the capnp tools (and the go-capnp "templates"), I already included the generated go files (from the .capnp).
We also need to generalize this comment so that it also works if the go-capnp "templates" are installed/downloaded to another location (maybe using an environment variable which is then set in the Makefile depending on which machine it runs).
Note that it is really important that the compile command is not invoked twice (once for each file) but only once (so that it is aware of both files when running).

About union/sub types
To avoid the need for one channel (x <> horizontalAPI) per type that is being sent over the horizontalAPI, we only use one channel on which we transfer basically an union type. Since golang sadly has no support for union/sum types, we use an interface and then use switch x.(type) to check what type has been sent. When adding a new type this approach is prone to errors (forgetting to extend the switch statement). Therefore we make use of the go-sumtype tool to mitigate (core to this concept is that the interface hast at least one unexported/lowercase function). Also see the new target in the Makefile.
(we should make use of this in the verticalAPI as well -> refactoring step for later)




