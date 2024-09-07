# How to run/build (dockerized):

## Build the docker image

```bash
docker build -t gossip-3 .
```

## Start a container for each node

Assumptions:
- you want to bring up 3 nodes (change the for loop accordingly otherwise)
- you have a config `.ini` file to configure each node which is named `node<nodeNr>.ini` and is placed in the current working directory.
- note that your config should not configure the horizontal API (`6001`) as well as the vertical API (`7001`) to different ports (the dockerfile does not expose them otherwise)

```bash
# create a network to run the nodes in so that
# the nodes can communicate with each other
docker network create gossipNet
for node in 0 1 2 3 ; do
    docker run --network gossipNet -d --name "node${node}" -v "$(pwd)/node${node}.ini":/config.ini -p $(( 7000+node )):7001 -p $(( 6000+node )):6001 gossip-3 -c /config.ini
done
```

> [!Note]
> Docker will assign each node an IP Address. Since you might not know this in
> advance you can also simply use the name you assigned to that container when
> specifying the neighbors in the config file.

For an example on how to setup a simple dockerized two node network and use
voidphone_pytest to send an announcement see our small
[tutorial](tutorial/README.md).

# How to run tests
## golang tests
You can run
```bash
make test
```
to run all the unit- and end-to-end tests (for all golang
packages) we've written. Note that this will take some time because of the
end-to-end tests.

> [!Note]
> Since we already had all the tests (including the end-to-end tests ready
> before the requirements changed that we should supply a dockerfile, our tests
> do not run dockerized. For more information on our testing setup see our
> documentstion.
> We explicitly asked if our approach (of not dockerizing our existing tests) is
> ok on [moodle](TODO).

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
