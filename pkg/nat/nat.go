package nat

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/pkg/network"
)

type Role string

const (
	ServerSide Role = "server"
	ClientSide Role = "client"

	maxRepeatTimes      = 5
	waitMakeHoleTimeout = 5 * time.Second
)

type Nat struct {
	LocalAddrs        []*net.UDPAddr
	RemoteLocalAddrs  []*net.UDPAddr
	RemoteMappedAddrs []*net.UDPAddr
	Role              Role
	Resource          Resource
	Actions           []Action
}

func MakeHole(t *Nat) (conn net.PacketConn, raddr *net.UDPAddr, err error) {
	localConns, remoteAddrs := genEndpoint(t)

	for _, action := range t.Actions {
		if action.Wait > 0 {
			time.Sleep(action.Wait)
		}
		if action.TryRemote {
			send(action.LowTTL, localConns, remoteAddrs)
		}

		repeatTimes := maxRepeatTimes
		if !action.Repeat {
			repeatTimes = 1
		}

		for i := 0; i < repeatTimes; i++ {
			send(action.LowTTL, localConns, remoteAddrs)
			conn, raddr, err := wait(localConns, waitMakeHoleTimeout)
			if err != nil {
				logrus.Debugf("wait for reply err, %s", err.Error())
				continue
			}
			logrus.Debugf("wait for reply success")

			if action.TryRemote {
				echoTo(conn, raddr)
			}
			return conn, raddr, nil
		}
	}
	return conn, raddr, fmt.Errorf("make hole error")
}

func echoTo(conn *net.UDPConn, raddr *net.UDPAddr) error {
	_, err := conn.WriteToUDP([]byte("a"), raddr)
	return err
}

func echo(conn *net.UDPConn) error {
	_, err := conn.Write([]byte("a"))
	return err
}

func send(lowTTL bool, localConns []*net.UDPConn, remoteAddrs []*net.UDPAddr) {
	sendPair := func(lowTTL bool, local *net.UDPConn, remote *net.UDPAddr) (err error) {
		if lowTTL {
			err = network.ModifyTTL(local, 8)
			if err != nil {
				return err
			}
		}
		_, err = local.WriteToUDP([]byte("a"), remote)
		return err
	}
	logrus.Debugf("send len(%d) -> len(%d)", len(localConns), len(remoteAddrs))
	for _, local := range localConns {
		for _, remote := range remoteAddrs {
			err := sendPair(lowTTL, local, remote)
			if err != nil {
				logrus.Debugf("send udp error, %s", err.Error())
			}
		}
	}
}

func wait(localConns []*net.UDPConn, timeout time.Duration) (conn *net.UDPConn, addr *net.UDPAddr, err error) {
	type node struct {
		conn *net.UDPConn
		addr *net.UDPAddr
	}
	recvCh := make(chan node)

	wg := sync.WaitGroup{}
	waitForReply := func(local *net.UDPConn) {
		defer wg.Done()
		local.SetReadDeadline(time.Now().Add(timeout))
		defer local.SetReadDeadline(time.Time{})
		p := make([]byte, 4)
		_, a, err := local.ReadFromUDP(p)
		if err != nil {
			return
		}
		result := node{
			conn: local,
			addr: a,
		}
		recvCh <- result
	}

	for _, local := range localConns {
		wg.Add(1)
		go waitForReply(local)
	}

	go func() {
		wg.Wait()
		close(recvCh)
	}()
	pair, ok := <-recvCh
	if !ok {
		return nil, nil, errTimedOut
	}
	return pair.conn, pair.addr, nil
}

func genEndpoint(t *Nat) (localConns []*net.UDPConn, remoteAddrs []*net.UDPAddr) {
	localConns = make([]*net.UDPConn, 0)

	for _, l := range t.LocalAddrs {
		c, err := net.ListenUDP("udp", l)
		if err != nil {
			logrus.Infof("listen %s:%d failed, %s", l.IP.String(), l.Port, err.Error())
			continue
		}
		localConns = append(localConns, c)
	}

	for i := 1; i < t.Resource.LocalPortCount; i++ {
		c, err := net.ListenUDP("udp", nil)
		if err != nil {
			logrus.Infof("listen new port failed, %s", err.Error())
			continue
		}
		localConns = append(localConns, c)
	}

	remoteAddrs = make([]*net.UDPAddr, 0)
	remoteAddrs = append(remoteAddrs, t.RemoteMappedAddrs...)
	if t.Resource.RemotePortCount > 1 {
		additionRemotePorts := NoRepeatRandInts(
			t.Resource.RemotePortStart,
			t.Resource.RemotePortEnd,
			t.Resource.RemotePortCount,
		)
		ips := make([]net.IP, 0)
		for _, a := range t.RemoteMappedAddrs {
			ips = append(ips, a.IP)
		}
		for _, port := range additionRemotePorts {
			ip := RandomOne(ips)
			remoteAddrs = append(remoteAddrs, &net.UDPAddr{IP: ip, Port: port})
		}
	}

	return localConns, remoteAddrs
}

func bind(conn *net.UDPConn, addr *net.UDPAddr) (c *net.UDPConn, err error) {
	local, err := net.ResolveUDPAddr("udp", conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}
	err = conn.Close()
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", local, addr)
}
