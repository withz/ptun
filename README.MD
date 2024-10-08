# Introduction

This tool can build a local area network with TUN device, and transfer the net traffic by using P2P connection.

This tool has two parts. One is hub, which installed on cloud server with PUBLIC IP. It can exchange nodes' info and resolve nodes' NAT public ip. The other one is node, which installed on client and create TUN interface on machine. When P2P connection established, TUN device will listen the TUN IP and transfer the traffic to P2P connection.

# Build

```shell
make all
```

# Usage

On Hub(server side, which has PUBLIC IP)
```shell
./hub -c ptun-hub.toml
```

On Node1(client side, which needs p2p connect). This will create TUN interface with IP net 192.168.58.11/24.
```shell
sudo ./node -c ptun-node1.toml
```

On Node2(client side, which needs p2p connect). This will create TUN interface with IP net 192.168.58.12/24.
```shell
sudo ./node -c ptun-node2.toml
```

# Speed Test

Speed test result:
```shell
-----------------------------------------------------------
Server listening on 5201 (test #1)
-----------------------------------------------------------
Accepted connection from 192.168.58.12, port 51824
[  5] local 192.168.58.11 port 5201 connected to 192.168.58.12 port 51826
[ ID] Interval           Transfer     Bitrate
[  5]   0.00-1.00   sec  46.1 MBytes   387 Mbits/sec                  
[  5]   1.00-2.00   sec  47.8 MBytes   401 Mbits/sec                  
[  5]   2.00-3.00   sec  45.5 MBytes   382 Mbits/sec                  
[  5]   3.00-4.00   sec  45.5 MBytes   381 Mbits/sec                  
[  5]   4.00-5.00   sec  43.1 MBytes   362 Mbits/sec                  
[  5]   5.00-6.00   sec  44.4 MBytes   372 Mbits/sec                  
[  5]   6.00-7.00   sec  48.2 MBytes   405 Mbits/sec                  
[  5]   7.00-8.00   sec  46.2 MBytes   388 Mbits/sec                  
[  5]   8.00-9.00   sec  45.5 MBytes   382 Mbits/sec                  
[  5]   9.00-10.00  sec  47.4 MBytes   398 Mbits/sec                  
[  5]  10.00-10.00  sec   128 KBytes   396 Mbits/sec                  
- - - - - - - - - - - - - - - - - - - - - - - - -
[ ID] Interval           Transfer     Bitrate
[  5]   0.00-10.00  sec   460 MBytes   386 Mbits/sec                  receiver
```