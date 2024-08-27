# How tu run/build (dockerized):

## Build the docker image

```bash
docker build -t gossip-3 .
```

## Start a container for each node

Assumptions:
- you want to bring up 3 nodes (change the for loop accordingly otherwise)
- you have a config `.ini` file to configure each node which is named `node<nodeNr>.ini` and is placed in the current working directory.

```bash
for node in 0 1 2 3 ; do
    docker run -d --name "node${node}" -v "$(pwd)/node${node}.ini":/config.ini -p 7001:7001 -p 6001:6001 gossip-3 -c /config.ini
done
```

# How to run tests
## golang tests
You can run
```bash
make test
```
to run all the unit and end-to-end tests (for all golang
packages) we've written. Note that this will take some time because of the
end-to-end tests.

## staticcheck
We're using the tool `staticcheck` (install via `go install
honnef.co/go/tools/cmd/staticcheck@latest`) for static analysis. You can simply
run these checks via
```bash
make staticcheck
```

## sumtype
We are using the sumtype idiom to model union/sumtype-like behaviour. In the
golang code this reflects in the use of type switch-case statements. In order to
check if all necessary cases are handled we are using te `go-sumtype` tool
(install via `go install github.com/BurntSushi/go-sumtype@latest`). You can run
this check via

```bash
make checkUnion
````
