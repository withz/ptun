package app

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/pkg/bridge"
	"github.com/withz/ptun/pkg/device"
	"github.com/withz/ptun/pkg/nat"
	"github.com/withz/ptun/pkg/network"
	"github.com/withz/ptun/pkg/proto"
)

type Route struct {
	Next    net.IP
	Network *net.IPNet
}

type P2PNetworkConfig struct {
	Tun       string
	IP        string
	AllowNets []string
	Routers   []struct {
		Next     string
		Networks []string
	}
}

type P2PNetwork struct {
	bridge    *bridge.Bridge
	rules     *device.RuleManager
	routes    []*Route
	peerMutex sync.Mutex
}

func CreateNet(cfg *P2PNetworkConfig) (*P2PNetwork, error) {
	_, ipnet, err := net.ParseCIDR(cfg.IP)
	if err != nil {
		return nil, fmt.Errorf("create p2p network err, %w", err)
	}
	peerRoutes := make([]*Route, 0)
	vethRoutes := make([]string, 0)
	for _, r := range cfg.Routers {
		i := net.ParseIP(r.Next)
		for _, n := range r.Networks {
			_, ipnet, err := net.ParseCIDR(n)
			if err != nil {
				return nil, fmt.Errorf("create p2p network err, %w", err)
			}
			peerRoutes = append(peerRoutes, &Route{Next: i, Network: ipnet})
		}
		vethRoutes = append(vethRoutes, r.Networks...)
	}

	veth, err := device.NewTun(cfg.Tun, []string{cfg.IP}, vethRoutes)
	if err != nil {
		return nil, fmt.Errorf("p2p network create veth err, %w", err)
	}
	bdg := bridge.NewBridge(veth)

	routes := make([]*net.IPNet, 0)
	for _, r := range cfg.AllowNets {
		_, ipnet, err := net.ParseCIDR(r)
		if err != nil {
			return nil, fmt.Errorf("create p2p network err, %w", err)
		}
		routes = append(routes, ipnet)
	}
	ipt, err := device.NewRuleManager()
	if err != nil {
		return nil, fmt.Errorf("create p2p network err, %w", err)
	}
	// clear iptables on exit
	err = ipt.UpdateIptables(ipnet.String(), routes)
	if err != nil {
		return nil, fmt.Errorf("create p2p network err, %w", err)
	}
	return &P2PNetwork{
		bridge: bdg,
		rules:  ipt,
		routes: peerRoutes,
	}, nil
}

func (nw *P2PNetwork) HasPeer(name string) bool {
	nw.peerMutex.Lock()
	defer nw.peerMutex.Unlock()
	return nw.bridge.HasPeer(name)
}

func (nw *P2PNetwork) NewNatPeer(name string, remoteIp string, token string, m *nat.Nat) error {
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
	for _, r := range nw.routes {
		if r.Next.Equal(remoteIP) {
			routes = append(routes, r.Network)
		}
	}

	peer := bridge.NewPeer(name, []*net.IPNet{remoteIPNet}, routes, proto.NewTransport(econn))
	return nw.bridge.ConnectPeer(peer)
}

func (nw *P2PNetwork) OnShutdown() {
	nw.rules.ClearAllRules()
}
