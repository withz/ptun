package nat

import (
	"errors"
	"net"
	"slices"
	"time"

	"github.com/pion/stun/v2"
)

type Behavior string

const (
	NoNAT                Behavior = "NoNAT"
	EndpointIndependent  Behavior = "EndpointIndependent"
	AddressDependent     Behavior = "AddressDependent"
	AddressPortDependent Behavior = "AddressPortDependent"
	Unknown              Behavior = "Unknown"
	RegularPort          Behavior = "RegularPort"
	RandomPort           Behavior = "RandomPort"
)

var (
	errResponseMessage = errors.New("error reading from response message channel")
	errTimedOut        = errors.New("timed out waiting for response")
	errNoOtherAddress  = errors.New("no OTHER-ADDRESS in message")
	errTestFailure     = errors.New("test failure")
)

type StunBehavior struct {
	MappingBehavior Behavior
	FilterBehavior  Behavior

	LocalAddrs     []*net.UDPAddr
	MappedAddrs    []*net.UDPAddr
	MappedIpList   []string
	MappedPortList []int
}

func AnalyzeStunBehavior(stunAddr string) (*StunBehavior, error) {
	conn, err := connectStun(stunAddr, nil)
	if err != nil {
		return nil, err
	}
	defer conn.conn.Close()

	mappingBehavior, err := mappingTests(conn)
	if err != nil {
		return nil, err
	}

	filterBehavior, err := filterTests(conn)
	if err != nil {
		return nil, err
	}

	if mappingBehavior == Unknown || filterBehavior == Unknown {
		return nil, errTestFailure
	}

	localAddrs := make([]*net.UDPAddr, 0)
	localAddrs = append(localAddrs, conn.LocalAddr)

	ipList, portList := make([]string, 0), make([]int, 0)
	for _, addr := range conn.mappedAddrs {
		ip, port := addr.IP.String(), addr.Port

		if !slices.Contains(ipList, ip) {
			ipList = append(ipList, addr.IP.String())
		}
		if !slices.Contains(portList, port) {
			portList = append(portList, addr.Port)
		}
	}

	return &StunBehavior{
		MappingBehavior: mappingBehavior,
		FilterBehavior:  filterBehavior,
		LocalAddrs:      localAddrs,
		MappedAddrs:     conn.mappedAddrs,
		MappedIpList:    ipList,
		MappedPortList:  portList,
	}, nil
}

func MappingTests(stunAddr string) (Behavior, error) {
	conn, err := connectStun(stunAddr, nil)
	if err != nil {
		return Unknown, err
	}
	defer conn.conn.Close()

	return mappingTests(conn)
}

func FilterTests(stunAddr string) (Behavior, error) {
	conn, err := connectStun(stunAddr, nil)
	if err != nil {
		return Unknown, err
	}
	defer conn.conn.Close()

	return filterTests(conn)
}

func mappingTests(conn *stunServerConn) (Behavior, error) {
	// Test 1,
	//   LocalAddr == XorAddr, No NAT
	//   Else, NAT
	test1, err := mappingTest1(conn)
	if err != nil {
		return Unknown, err
	}
	if test1.xorAddr.String() == conn.LocalAddr.String() {
		return NoNAT, nil
	}

	// Test 2, change STUN server IP,
	//   XorAddr not changed, Endpoint Independent
	//   XorAddr changed, X Dependent
	test2, err := mappingTest2(conn)
	if err != nil {
		return Unknown, err
	}
	if test2.xorAddr.String() == test1.xorAddr.String() {
		return EndpointIndependent, nil
	}

	// Test 3, change STUN server Port,
	//   XorAddr not changed, Address Dependent
	//   XorAddr changed, Address Port Dependent
	test3, err := mappingTest3(conn)
	if err != nil {
		return Unknown, err
	}
	if test3.xorAddr.String() == test2.xorAddr.String() {
		return AddressDependent, nil
	} else {
		return AddressPortDependent, nil
	}
}

func mappingTest1(conn *stunServerConn) (*stunResult, error) {
	req, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return nil, err
	}
	conn.request = req
	resp, err := conn.Do(req, conn.RemoteAddr)
	if err != nil {
		return nil, err
	}
	result := parseStunMessage(resp)
	if result.otherAddr == nil || result.xorAddr == nil {
		return nil, errNoOtherAddress
	}
	addr, err := net.ResolveUDPAddr("udp4", result.otherAddr.String())
	if err != nil {
		return nil, err
	}
	conn.OtherAddr = addr

	mappedAddr := &net.UDPAddr{
		IP:   net.ParseIP(result.xorAddr.IP.String()),
		Port: result.xorAddr.Port,
	}
	conn.mappedAddrs = append(conn.mappedAddrs, mappedAddr)
	return result, nil
}

func mappingTest2(conn *stunServerConn) (*stunResult, error) {
	req := conn.request
	oaddr := *conn.OtherAddr
	oaddr.Port = conn.RemoteAddr.Port
	resp, err := conn.Do(req, &oaddr)
	if err != nil {
		return nil, err
	}

	result := parseStunMessage(resp)
	if result.xorAddr != nil {
		mappedAddr := &net.UDPAddr{
			IP:   net.ParseIP(result.xorAddr.IP.String()),
			Port: result.xorAddr.Port,
		}
		conn.mappedAddrs = append(conn.mappedAddrs, mappedAddr)
	}
	return result, nil
}

func mappingTest3(conn *stunServerConn) (*stunResult, error) {
	req := conn.request
	resp, err := conn.Do(req, conn.OtherAddr)
	if err != nil {
		return nil, err
	}
	result := parseStunMessage(resp)
	if result.xorAddr != nil {
		mappedAddr := &net.UDPAddr{
			IP:   net.ParseIP(result.xorAddr.IP.String()),
			Port: result.xorAddr.Port,
		}
		conn.mappedAddrs = append(conn.mappedAddrs, mappedAddr)
	}
	return result, nil
}

func filterTests(conn *stunServerConn) (Behavior, error) {
	// Test 1, Connection check
	_, err := filterTest1(conn)
	if err != nil {
		return Unknown, err
	}

	// Test 2, Request to change IP & Port
	b, err := filterTest2(conn)
	if err != nil {
		return Unknown, err
	}
	if b != Unknown {
		return b, nil
	}

	// Test 3, Request to change Port
	b, err = filterTest3(conn)
	if err != nil {
		return Unknown, err
	}
	return b, nil
}

func filterTest1(conn *stunServerConn) (Behavior, error) {
	req, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return Unknown, err
	}
	resp, err := conn.Do(req, conn.RemoteAddr)
	if err != nil {
		return Unknown, err
	}
	result := parseStunMessage(resp)
	if result.xorAddr == nil || result.otherAddr == nil {
		return Unknown, errNoOtherAddress
	}
	addr, err := net.ResolveUDPAddr("udp4", result.otherAddr.String())
	if err != nil {
		return Unknown, err
	}
	conn.OtherAddr = addr

	mappedAddr := &net.UDPAddr{
		IP:   net.ParseIP(result.xorAddr.IP.String()),
		Port: result.xorAddr.Port,
	}
	conn.mappedAddrs = append(conn.mappedAddrs, mappedAddr)
	return Unknown, nil
}

func filterTest2(conn *stunServerConn) (Behavior, error) {
	req, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return Unknown, err
	}
	req.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x06})
	resp, err := conn.Do(req, conn.RemoteAddr)
	if err == nil {
		result := parseStunMessage(resp)
		if result.xorAddr != nil {
			mappedAddr := &net.UDPAddr{
				IP:   net.ParseIP(result.xorAddr.IP.String()),
				Port: result.xorAddr.Port,
			}
			conn.mappedAddrs = append(conn.mappedAddrs, mappedAddr)
		}
		return EndpointIndependent, nil
	} else if errors.Is(err, errTimedOut) {
		return Unknown, nil
	}

	return Unknown, err
}

func filterTest3(conn *stunServerConn) (Behavior, error) {
	req, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return Unknown, err
	}
	req.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x02})
	resp, err := conn.Do(req, conn.RemoteAddr)
	if err == nil {
		result := parseStunMessage(resp)
		if result.xorAddr != nil {
			mappedAddr := &net.UDPAddr{
				IP:   net.ParseIP(result.xorAddr.IP.String()),
				Port: result.xorAddr.Port,
			}
			conn.mappedAddrs = append(conn.mappedAddrs, mappedAddr)
		}
		return AddressDependent, nil
	} else if errors.Is(err, errTimedOut) {
		return AddressPortDependent, nil
	}

	return Unknown, err
}

type stunServerConn struct {
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr
	OtherAddr  *net.UDPAddr

	conn        net.PacketConn
	messageChan chan *stun.Message
	request     *stun.Message
	mappedAddrs []*net.UDPAddr
}

func connectStun(stunAddr string, laddr *net.UDPAddr) (*stunServerConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", stunAddr)
	if err != nil {
		return nil, err
	}
	c, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, err
	}

	mChan := make(chan *stun.Message)

	go func() {
		for {
			buf := make([]byte, 1024)
			n, _, err := c.ReadFromUDP(buf)
			if err != nil {
				close(mChan)
				return
			}
			buf = buf[:n]

			m := new(stun.Message)
			m.Raw = buf
			err = m.Decode()
			if err != nil {
				close(mChan)
				return
			}
			mChan <- m
		}
	}()

	return &stunServerConn{
		conn:        c,
		LocalAddr:   c.LocalAddr().(*net.UDPAddr),
		RemoteAddr:  addr,
		messageChan: mChan,
		mappedAddrs: make([]*net.UDPAddr, 0),
	}, nil
}

func (s *stunServerConn) Do(req *stun.Message, addr net.Addr) (*stun.Message, error) {
	err := req.NewTransactionID()
	if err != nil {
		return nil, err
	}
	_, err = s.conn.WriteTo(req.Raw, addr)
	if err != nil {
		return nil, err
	}

	select {
	case m, ok := <-s.messageChan:
		if !ok {
			return nil, errResponseMessage
		}
		return m, nil
	case <-time.After(time.Duration(3000) * time.Millisecond):
		return nil, errTimedOut
	}
}

type stunResult struct {
	xorAddr    *stun.XORMappedAddress
	otherAddr  *stun.OtherAddress
	respOrigin *stun.ResponseOrigin
	mappedAddr *stun.MappedAddress
	software   *stun.Software
}

func parseStunMessage(msg *stun.Message) *stunResult {
	result := &stunResult{}
	result.mappedAddr = &stun.MappedAddress{}
	result.xorAddr = &stun.XORMappedAddress{}
	result.respOrigin = &stun.ResponseOrigin{}
	result.otherAddr = &stun.OtherAddress{}
	result.software = &stun.Software{}
	if result.xorAddr.GetFrom(msg) != nil {
		result.xorAddr = nil
	}
	if result.otherAddr.GetFrom(msg) != nil {
		result.otherAddr = nil
	}
	if result.respOrigin.GetFrom(msg) != nil {
		result.respOrigin = nil
	}
	if result.mappedAddr.GetFrom(msg) != nil {
		result.mappedAddr = nil
	}
	if result.software.GetFrom(msg) != nil {
		result.software = nil
	}
	return result
}
