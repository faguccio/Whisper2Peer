# Types
## Bandwidth
Not interesting, is it?

## Latency
- time to silence? (fixed TTL, look at it distance whise, start/stop? (duration or absolute time?))

## Loss Prob / Coverage
- %nodes reached vs distance (fixed TTL)

## Amount of Packets
- total value
- in relation to TTL
- Maybe come up with expected value (e.g. by drawing something like a spanning tree) for very specific scenario

pkg/node = #rounds

nodes reached/node = N/N + N-1/N + ... + N-R/N
(not quite since maybe not all neighbors are still uninfected)

# Comparison
How can this be compared with other graphs? This would have other nodes and edges => would maybe need to average over multiple randomly selected nodes (but average sometimes can be hard) to make the measurement independant of the selected source node. Could potentially do this at the same time (select x nodes to which send announcement at the same time)

# TODOs
- run `sar` when measuring (in Makefile) to monitor device stats (cpu/io bound)
- testing script should take prefix as argument for generated files (check `len(os.Argv)` at beginning)

## testutils
- get rid of sleep in setup (get notified via channel?)
- log goroutine: send signal if packet on hzApi was observed
- `tester.WaitUntilSilent(context, [gtype])` using above signal to wait until 2 round times silence (context can be used to apply a deadline)
- registerTypre for all peers
- filter for gtype (in processing)

https://github.com/jszwec/csvutil/issues/50
