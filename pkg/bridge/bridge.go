package bridge

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/pkg/network"
)

type Veth interface {
	io.ReadWriter
}

type Bridge struct {
	peers sync.Map
	veth  Veth
	pool  *sync.Pool
}

func NewBridge(veth Veth) *Bridge {
	bdg := &Bridge{
		veth: veth,
		pool: &sync.Pool{
			New: func() any {
				p := make([]byte, 8192)
				return &p
			},
		},
	}
	bdg.handleVeth()
	return bdg
}

func (b *Bridge) ConnectPeer(p *Peer) error {
	old, ok := b.getPeer(p.name)
	if ok {
		b.DisconnectPeer(old)
	}
	err := b.addPeer(p)
	go b.handlePeer(p)
	return err
}

func (b *Bridge) DisconnectPeer(p *Peer) {
	p.Close()
	b.delPeer(p.name)
}

func (b *Bridge) handlePeer(p *Peer) {
	p.SetKeepalive(10 * time.Second)
	for {
		v := b.pool.Get().(*[]byte)
		n, err := p.Read(*v)
		if err != nil {
			logrus.Debugf("handle peer err, %s", err.Error())
			b.DisconnectPeer(p)
			return
		}

		_, s, d := network.ParsePacket(*v)
		src := net.IP(s)
		dst := net.IP(d)
		logrus.Tracef("Peer: %s -> %s", src.String(), dst.String())

		b.veth.Write((*v)[:n])
		b.pool.Put(v)
	}
}

func (b *Bridge) handleVeth() {
	go func() {
		for {
			p := b.pool.Get().(*[]byte)
			n, err := b.veth.Read(*p)
			if err != nil {
				logrus.Debugf("read veth err, %s", err.Error())
				continue
			}
			data := (*p)[:n]
			_, s, d := network.ParsePacket(data)
			src := net.IP(s)
			dst := net.IP(d)

			// route
			if network.IsBroadcast(dst) {
				b.peers.Range(func(key, value any) bool {
					p := value.(*Peer)
					p.Write(data)
					logrus.Tracef("Veth: %s -> %s", src.String(), dst.String())
					return true
				})
			} else {
				b.peers.Range(func(key, value any) bool {
					p := value.(*Peer)
					if p.HasIP(dst.String()) {
						p.Write(data)
						logrus.Tracef("Veth: %s -> %s", src.String(), dst.String())
						return true
					}
					return true
				})
			}
			b.pool.Put(p)
		}
	}()
}

func (b *Bridge) Peers() []*Peer {
	peers := []*Peer{}
	b.peers.Range(func(key, value any) bool {
		peers = append(peers, value.(*Peer))
		return true
	})
	return peers
}

func (b *Bridge) HasPeer(name string) bool {
	_, ok := b.getPeer(name)
	return ok
}

func (b *Bridge) getPeer(name string) (*Peer, bool) {
	v, ok := b.peers.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*Peer), true
}

func (b *Bridge) addPeer(p *Peer) error {
	_, ok := b.getPeer(p.name)
	if ok {
		return fmt.Errorf("peer already exist, remove it first")
	}
	b.peers.Store(p.name, p)
	return nil
}

func (b *Bridge) delPeer(name string) (*Peer, error) {
	p, ok := b.getPeer(name)
	if !ok {
		return nil, fmt.Errorf("peer not exist")
	}
	b.peers.Delete(name)
	return p, nil
}
