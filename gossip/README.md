# How tu run/build (dockerized):

## Build the docker image

`docker build -t gossip-3 .`

## Start a container for each node

Assumptions:
- you want to bring up 3 nodes (change the for loop accordingly otherwise)
- you have a config `.ini` file to configure each node which is named `node<nodeNr>.ini` and is placed in the current working directory.

```bash
for node in 0 1 2 3 ; do
    docker run -d --name "node${node}" -v "$(pwd)/node${node}.ini":/config.ini -p 7001:7001 -p 6001:6001 gossip-3 -c /config.ini
done
```
