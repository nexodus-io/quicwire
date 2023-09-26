# quicwire

> **Warning**: This is a work in progress and purely experimental.

It's an attempt to implement a wireguard like tunneled mesh network using QUIC protocol.

## Build quicwire binary and run it

```bash
make build
```

## Configuration

Update the sample config file present [here](./hack/sample.conf). If you attempted to do tunneling with wireguard, this format should be familiar to you.

```text
[Interface]
# This IP address will be assigned to the local tunnel interface
LocalEndpoint = 10.100.0.1
# Local Node IP address on which the server will listen for incoming connection
LocalNodeIp = xxx.xxx.xxx.xxx
# Port on which the server will listen for incoming connections
ListenPort = 55380

[Peer]
# Tunnel IP address assigned to the peer by it's agent
AllowedIPs = 10.100.0.2
# Reflexive IP address of the Peer
Endpoint = xxx.xxx.xxx.xxx:55380
# Keep alive interval for QUIC connection
PersistentKeepalive = 10

```

## Connect two nodes with QUIC tunnel

Run the following command on each node with it's respective config file.

```bash
./dist/qw --config-file hack/<update_conf_file.conf>
```

You need to update the sample file for each of the node that you want to connect to this mesh network. If you have more than one peer to connect to, add [Peer] section per peer in the config file.

## Utilities

### Stun-client

If you would like to find the reflexive address of the node, you can use the utility present in `hack/stun-client`. This is a simple stun client that will send a stun request to the server and print the reflexive address of the node.

#### Build the stun-client binary

```bash
build-stun
```

#### Run the stun-client 

Run the stun client binary to find the reflexive address (e.g., the public address portion of your NAT binding at the point your device as seen from the STUN server perspective)

```bash
./dist/stun-client --source-port 55380 -stun-server stun1.l.google.com:19302
```

It should output the reflexive address and the port number used by your NAT device to forward the request to the server.

```text
INFO[0000] Stun request to [stun1.l.google.com:19302]:55380 result is: 44.203.3.88:55380
```

You can use this information to update the Peer's Endpoint field in the config file.
