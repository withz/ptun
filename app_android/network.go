package app

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/pkg/bridge"
	"github.com/withz/ptun/pkg/nat"
	"github.com/withz/ptun/pkg/network"
	"github.com/withz/ptun/pkg/proto"
)

type route struct {
	Next    net.IP
	Network *net.IPNet
}

type P2PNetworkConfig struct {
	Routers []*route
}

func (c *P2PNetworkConfig) AddRoute(next string, dest string) bool {
	if c.Routers == nil {
		c.Routers = []*route{}
	}
	n := net.ParseIP(next)
	_, d, err := net.ParseCIDR(dest)
	if err != nil {
		return false
	}
	c.Routers = append(c.Routers, &route{
		Next:    n,
		Network: d,
	})
	return true
}

type P2PNetwork struct {
	cfg       *P2PNetworkConfig
	bridge    *bridge.Bridge
	peerMutex sync.Mutex
}

func CreateNet(cfg *P2PNetworkConfig) *P2PNetwork {
	if cfg == nil {
		return nil
	}

	veth := newAndroidVeth()
	bdg := bridge.NewBridge(veth)
	return &P2PNetwork{
		cfg:    cfg,
		bridge: bdg,
	}
}

func (nw *P2PNetwork) hasPeer(name string) bool {
	nw.peerMutex.Lock()
	defer nw.peerMutex.Unlock()
	return nw.bridge.HasPeer(name)
}

func (nw *P2PNetwork) newNatPeer(name string, remoteIp string, token string, m *nat.Nat) error {
	nw.peerMutex.Lock()
	defer nw.peerMutex.Unlock()
	remoteIP, remoteIPNet, err := net.ParseCIDR(remoteIp)
	if err != nil {
		return fmt.Errorf("parse ip err, %w", err)
	}
	remoteIPNet.IP = remoteIP

	conn, raddr, err := nat.MakeHole(m)
	if err != nil {
		return fmt.Errorf("make hole err, %w", err)
	}

	var econn net.Conn
	if true {
		econn, err = network.NewRawConn(conn, raddr)
		if err != nil {
			return fmt.Errorf("raw conn err, %w", err)
		}
	} else {
		econn, err = network.NewQuicConn(context.TODO(), conn, raddr, string(m.Role))
		if err != nil {
			return fmt.Errorf("init quic err, %w", err)
		}
		econn, err = network.NewEncyptedConn(econn, []byte(token))
		if err != nil {
			return fmt.Errorf("encypted err, %w", err)
		}
	}

	logrus.Infof("make hole success, wait connect. %v -> %v", conn.LocalAddr(), raddr)

	routes := make([]*net.IPNet, 0)
	for _, r := range nw.cfg.Routers {
		if r.Next.Equal(remoteIP) {
			routes = append(routes, r.Network)
		}
	}

	peer := bridge.NewPeer(name, []*net.IPNet{remoteIPNet}, routes, proto.NewTransport(econn))
	return nw.bridge.ConnectPeer(peer)
}

func (nw *P2PNetwork) Shutdown() {
	nw.peerMutex.Lock()
	defer nw.peerMutex.Unlock()
	peers := nw.bridge.Peers()
	for _, peer := range peers {
		nw.bridge.DisconnectPeer(peer)
	}
}
