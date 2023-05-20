# quicmesh

> **Warning**: This is a work in progress and purely experimental.

It's an attempt to implement a wireguard like tunneled mesh network using QUIC protocol.

## Build quicmesh binary and run it

```bash
make build
```

## Update the sample config file present (here)[./hack/sample.conf]. If you attempted to do tunneling with wireguard, this format should be familiar to you.

```
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
./dist/quicmesh -config hack/<update_conf_file.conf>
```

You need to update the sample file for each of the node that you want to connect to this mesh network. If you have more than one peer to connect to, add [Peer] section per peer in the config file.
