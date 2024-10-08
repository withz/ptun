package network

import (
	"io"
	"net"
	"time"

	"github.com/fatedier/golib/crypto"
)

func NewCryptoReadWriter(rw io.ReadWriter, key []byte) (io.ReadWriter, error) {
	encReader := crypto.NewReader(rw, key)
	encWriter, err := crypto.NewWriter(rw, key)
	if err != nil {
		return nil, err
	}
	return struct {
		io.Reader
		io.Writer
	}{
		Reader: encReader,
		Writer: encWriter,
	}, nil
}

type encyptedConn struct {
	conn   net.Conn
	cypter io.ReadWriter
}

func NewEncyptedConn(conn net.Conn, key []byte) (net.Conn, error) {
	c, err := NewCryptoReadWriter(conn, key)
	if err != nil {
		return conn, err
	}
	return &encyptedConn{
		conn:   conn,
		cypter: c,
	}, nil
}

func (ec *encyptedConn) Read(b []byte) (n int, err error) {
	return ec.cypter.Read(b)
}

func (ec *encyptedConn) Write(b []byte) (n int, err error) {
	return ec.cypter.Write(b)
}

func (ec *encyptedConn) Close() error {
	return ec.conn.Close()
}

func (ec *encyptedConn) LocalAddr() net.Addr {
	return ec.conn.LocalAddr()
}

func (ec *encyptedConn) RemoteAddr() net.Addr {
	return ec.conn.RemoteAddr()
}

func (ec *encyptedConn) SetDeadline(t time.Time) error {
	return ec.conn.SetDeadline(t)
}

func (ec *encyptedConn) SetReadDeadline(t time.Time) error {
	return ec.conn.SetReadDeadline(t)
}

func (ec *encyptedConn) SetWriteDeadline(t time.Time) error {
	return ec.conn.SetWriteDeadline(t)
}
