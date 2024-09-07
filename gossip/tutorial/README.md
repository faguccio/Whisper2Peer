# Tutorial for using docker to bring up a simple two node network

You can follow the next steps along, the required config files are available in
this folder. When executing step 3 and 4, be sure that this is your working directory or that there is a copy of `node1.ini` and `node2.ini`.

1. Build the image like explained in [README.md](../README.md)
2. create a docker network: `docker network create gossipNet`
3. Bring up the first node `node=1 ; docker run --network gossipNet -d --name "node${node}" -v "$(pwd)/node${node}.ini":/config.ini -p $(( 7000+node )):7001 -p $(( 6000+node )):6001 gossip-3 -c /config.ini`
4. Bring up the second node `node=2 ; docker run --network gossipNet -d --name "node${node}" -v "$(pwd)/node${node}.ini":/config.ini -p $(( 7000+node )):7001 -p $(( 6000+node )):6001 gossip-3 -c /config.ini`
5. Go to your download of `voidphone_pytest`
6. Run `python3 gossip_client.py -d 0.0.0.0 -p 7001 -a`
7. Run `python3 gossip_client.py -d 0.0.0.0 -p 7002 -n`
