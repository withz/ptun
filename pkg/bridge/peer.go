package bridge

import (
	"net"

	"github.com/withz/ptun/pkg/proto"
)

type Peer struct {
	*proto.Transport
	name   string
	ips    []*net.IPNet
	routes []*net.IPNet
}

func NewPeer(name string, ips []*net.IPNet, routes []*net.IPNet, conn *proto.Transport) *Peer {
	return &Peer{
		Transport: conn,
		name:      name,
		ips:       ips,
		routes:    routes,
	}
}

func (p *Peer) HasIP(ip string) bool {
	dst := net.ParseIP(ip)
	for _, ip := range p.ips {
		if ip.Contains(dst) {
			return true
		}
	}
	for _, route := range p.routes {
		if route.Contains(dst) {
			return true
		}
	}
	return false
}
