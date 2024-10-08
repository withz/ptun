package network

import "net"

func NewRawConn(conn net.PacketConn, raddr *net.UDPAddr) (net.Conn, error) {
	conn.Close()
	laddr, err := net.ResolveUDPAddr("udp", conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", laddr, raddr)
}
