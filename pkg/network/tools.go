package network

import (
	"net"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func ModifyTTL(conn *net.UDPConn, minus int) error {
	addr, err := net.ResolveUDPAddr("udp", conn.LocalAddr().String())
	if err != nil {
		return err
	}
	if addr.IP.To4() != nil {
		c := ipv4.NewConn(conn)
		originTtl, err := c.TTL()
		if err != nil {
			return err
		}
		err = c.SetTTL(originTtl - minus)
		defer c.SetTTL(originTtl)
		if err != nil {
			return err
		}
	} else if addr.IP.To16() != nil {
		c := ipv6.NewConn(conn)
		originLimit, err := c.HopLimit()
		if err != nil {
			return err
		}
		err = c.SetHopLimit(originLimit - minus)
		defer c.SetHopLimit(originLimit)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsBroadcast(ip net.IP) bool {
	if ip.IsMulticast() {
		return true
	}
	ipv4 := ip.To4()
	if ipv4 != nil {
		return ipv4.Equal(net.IPv4bcast) || ipv4[3] == 255
	}
	return false
}

func ResolveUDPAddrs(addrs []string) ([]*net.UDPAddr, error) {
	results := make([]*net.UDPAddr, 0)
	for _, addr := range addrs {
		r, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

const (
	IPv4offsetTotalLength = 2
	IPv4offsetSrc         = 12
	IPv4offsetDst         = IPv4offsetSrc + net.IPv4len
)

const (
	IPv6offsetPayloadLength = 4
	IPv6offsetSrc           = 8
	IPv6offsetDst           = IPv6offsetSrc + net.IPv6len
	IPv6FixedHeaderLength   = 40
)

func ParsePacket(data []byte) (version byte, src []byte, dst []byte) {
	if len(data) == 0 {
		return 0, nil, nil
	}
	version = data[0] >> 4
	if version == 4 {
		src = data[IPv4offsetSrc : IPv4offsetSrc+net.IPv4len]
		dst = data[IPv4offsetDst : IPv4offsetDst+net.IPv4len]
	} else if version == 6 {
		src = data[IPv6offsetSrc : IPv6offsetSrc+net.IPv6len]
		dst = data[IPv6offsetDst : IPv6offsetDst+net.IPv6len]
	} else {
		version = 0
		src, dst = nil, nil
	}
	return version, src, dst
}

func PacketGetPayload(data []byte) (protocol int, payload []byte) {
	if len(data) == 0 {
		return 0, nil
	}
	version := data[0] >> 4
	if version == 4 {
		headerLength := (data[0] & 0xF) << 2
		return int(data[9]), data[headerLength:]
	} else if version == 6 {
		nextHeader := int(data[6])
		offset, nextHeader := skipIPv6ExtensionHeaders(data[IPv6FixedHeaderLength:], nextHeader)
		return nextHeader, data[IPv6FixedHeaderLength+offset:]
	}
	return 0, nil
}

// skipIPv6ExtensionHeaders 跳过所有IPv6扩展头部，返回跳过的字节数和最终的NextHeader
func skipIPv6ExtensionHeaders(mixPayload []byte, nextHeader int) (int, int) {
	offset := 0
	for {
		switch nextHeader {
		case 41: // 扩展头部：Encapsulated IPv6 Header
			fallthrough
		case 0, 43, 44, 50, 51, 60: // 扩展头部类型: Hop-by-Hop, Routing, Fragment, AH, ESP, Destination
			nextHeader = int(mixPayload[offset])
			headerLen := (int(mixPayload[offset+1]) + 1) * 8
			offset += headerLen
		default:
			// 6: upd, 17: tcp, 58: icmpv6
			return offset, nextHeader
		}
	}
}

func ParseIPNets(cidrs []string) ([]net.IP, []*net.IPNet, error) {
	nets := make([]*net.IPNet, 0)
	ips := make([]net.IP, 0)
	for _, route := range cidrs {
		ip, n, err := net.ParseCIDR(route)
		if err != nil {
			return nil, nil, err
		}
		nets = append(nets, n)
		ips = append(ips, ip)
	}
	return ips, nets, nil
}
