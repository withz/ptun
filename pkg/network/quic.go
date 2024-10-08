package network

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/sirupsen/logrus"
)

type quicConn struct {
	listener *quic.Listener
	client   quic.Connection
	stream   quic.Stream
}

func NewQuicConn(ctx context.Context, conn net.PacketConn, raddr net.Addr, role string) (net.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic"},
	}

	if role == "server" {
		cert := newRandomTLSKeyPair()
		tlsConfig.Certificates = []tls.Certificate{*cert}

		conn.Close()
		listener, err := quic.ListenAddr(conn.LocalAddr().String(), tlsConfig, nil)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("wait accept")
		client, err := listener.Accept(ctx)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("wait accept stream")
		stream, err := client.AcceptStream(ctx)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("wait return")
		return &quicConn{
			listener: listener,
			client:   client,
			stream:   stream,
		}, nil
	} else {
		tlsConfig.ServerName = raddr.String()
		client, err := quic.Dial(ctx, conn, raddr, tlsConfig, nil)
		if err != nil {
			return nil, err
		}
		stream, err := client.OpenStreamSync(ctx)
		if err != nil {
			return nil, err
		}
		return &quicConn{
			client: client,
			stream: stream,
		}, nil
	}
}

func (qc *quicConn) Read(b []byte) (n int, err error) {
	return qc.stream.Read(b)
}

func (qc *quicConn) Write(b []byte) (n int, err error) {
	return qc.stream.Write(b)
}

func (qc *quicConn) Close() error {
	defer func() {
		if qc.listener != nil {
			qc.listener.Close()
		}
	}()
	return qc.stream.Close()
}

func (qc *quicConn) LocalAddr() net.Addr {
	return qc.client.LocalAddr()
}

func (qc *quicConn) RemoteAddr() net.Addr {
	return qc.client.RemoteAddr()
}

func (qc *quicConn) SetDeadline(t time.Time) error {
	return qc.stream.SetDeadline(t)
}

func (qc *quicConn) SetReadDeadline(t time.Time) error {
	return qc.stream.SetReadDeadline(t)
}

func (qc *quicConn) SetWriteDeadline(t time.Time) error {
	return qc.stream.SetDeadline(t)
}

func newRandomTLSKeyPair() *tls.Certificate {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&key.PublicKey,
		key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tlsCert
}
