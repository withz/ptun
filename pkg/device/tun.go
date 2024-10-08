package device

import (
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/pkg/network"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	defaultTunMtu   = 1420
	fixHeaderLength = 16
)

type iface interface {
	Up() error
	Down() error

	AddAddr(addrs ...*net.IPNet) error
	DelAddr(addrs ...*net.IPNet) error
	AddRoute(addrs ...*net.IPNet) error
	DelRoute(addrs ...*net.IPNet) error
}

type Tun struct {
	name   string
	addrs  []string
	routes []string
	dev    tun.Device
	iface  iface

	readBufs [][]byte
	bufSizes []int
	count    int
	current  int

	closeOnce sync.Once
}

func NewTun(name string, addrs []string, routes []string) (*Tun, error) {
	ips, ipnets, err := network.ParseIPNets(addrs)
	if err != nil {
		return nil, err
	}

	_, routeNets, err := network.ParseIPNets(routes)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(ips); i++ {
		ipnets[i].IP = ips[i]
	}

	t := &Tun{
		name:    name,
		addrs:   addrs,
		routes:  routes,
		iface:   &tunIface{name: name},
		count:   0,
		current: 0,
	}

	dev, err := tun.CreateTUN(t.name, defaultTunMtu)
	if err != nil {
		return nil, err
	}
	t.dev = dev

	err = t.iface.AddAddr(ipnets...)
	if err != nil {
		return nil, err
	}

	batchSize := t.dev.BatchSize()
	t.readBufs = make([][]byte, batchSize)
	t.bufSizes = make([]int, batchSize)
	for i := range t.readBufs {
		t.readBufs[i] = make([]byte, 4096)
	}

	err = t.iface.Up()
	if err != nil {
		return nil, err
	}

	err = t.iface.AddRoute(routeNets...)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Tun) Close() (err error) {
	t.closeOnce.Do(func() {
		t.iface.Down()
		err = t.dev.Close()
	})
	return err
}

func (t *Tun) WriteRaw(data ...[]byte) error {
	newData := make([][]byte, len(data))
	for i, d := range data {
		newData[i] = make([]byte, fixHeaderLength+len(d))
		copy(newData[i][fixHeaderLength:], d[:])
	}
	_, err := t.dev.Write(newData, fixHeaderLength)
	return err
}

func (t *Tun) Write(data []byte) (int, error) {
	return len(data), t.WriteRaw(data)
}

func (t *Tun) Read(data []byte) (int, error) {
	if t.count == 0 {
		count, err := t.dev.Read(t.readBufs, t.bufSizes, 0)
		if err != nil {
			return 0, fmt.Errorf("read tun device err, %w", err)
		}
		t.count = count
	}
	if len(data) < t.bufSizes[t.current] {
		logrus.Debugf("read tun device err, size = %d, buf length too short", t.bufSizes[t.current])
	}
	copy(data, t.readBufs[t.current][:t.bufSizes[t.current]])
	t.current += 1

	if t.current >= t.count {
		t.current = 0
		t.count = 0
	}
	return t.bufSizes[t.current], nil
}
