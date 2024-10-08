package network

import (
	"net"
	"time"

	kcp "github.com/xtaci/kcp-go/v5"
)

type KcpConn struct {
	kcpConn *kcp.UDPSession
}

func NewKcpConn(conn net.PacketConn, raddr net.Addr) (net.Conn, error) {
	kcpConn, err := kcp.NewConn3(1, raddr, nil, 10, 3, conn)
	if err != nil {
		return nil, err
	}
	kcpConn.SetStreamMode(true)
	kcpConn.SetWriteDelay(true)
	kcpConn.SetNoDelay(1, 20, 2, 1)
	kcpConn.SetMtu(1350)
	kcpConn.SetWindowSize(1024, 1024)
	kcpConn.SetACKNoDelay(false)
	return &KcpConn{
		kcpConn: kcpConn,
	}, nil
}

func (k *KcpConn) Read(b []byte) (n int, err error) {
	return k.kcpConn.Read(b)
}

func (k *KcpConn) Write(b []byte) (n int, err error) {
	return k.kcpConn.Write(b)
}

func (k *KcpConn) Close() error {
	return k.kcpConn.Close()
}

func (k *KcpConn) LocalAddr() net.Addr {
	return k.kcpConn.LocalAddr()
}

func (k *KcpConn) RemoteAddr() net.Addr {
	return k.kcpConn.RemoteAddr()
}

func (k *KcpConn) SetDeadline(t time.Time) error {
	return k.kcpConn.SetDeadline(t)
}

func (k *KcpConn) SetReadDeadline(t time.Time) error {
	return k.kcpConn.SetReadDeadline(t)
}

func (k *KcpConn) SetWriteDeadline(t time.Time) error {
	return k.kcpConn.SetWriteDeadline(t)
}
