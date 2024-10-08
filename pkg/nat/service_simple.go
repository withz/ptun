package nat

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

type mappedInfo struct {
	MappedAddr string
}

type Server struct {
	PrimaryPort   int
	SecondaryPort int

	primaryListener   *net.UDPConn
	secondaryListener *net.UDPConn
}

func NewSimpleServer(p1, p2 int) *Server {
	s := &Server{
		PrimaryPort:   p1,
		SecondaryPort: p2,
	}
	return s
}

func (s *Server) Start() (err error) {
	logrus.Info("simple nat server start")

	s.primaryListener, err = net.ListenUDP("udp", &net.UDPAddr{Port: s.PrimaryPort})
	if err != nil {
		return err
	}

	s.secondaryListener, err = net.ListenUDP("udp", &net.UDPAddr{Port: s.SecondaryPort})
	if err != nil {
		return err
	}

	handler := func(conn *net.UDPConn) {
		for {
			err := s.handleConnection(conn)
			if err == io.EOF {
				logrus.Info("simple nat server stopped")
				break
			}
			if err != nil {
				logrus.Errorf("handle connection error, %s", err.Error())
				break
			}
		}
	}

	go handler(s.primaryListener)
	go handler(s.secondaryListener)

	return nil
}

func (s *Server) Stop() error {
	s.primaryListener.Close()
	s.secondaryListener.Close()
	return nil
}

func (s *Server) handleConnection(conn *net.UDPConn) error {
	p := make([]byte, 1024)
	_, addr, err := conn.ReadFromUDP(p)
	if err != nil {
		return err
	}
	result := &mappedInfo{
		MappedAddr: fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port),
	}
	p, err = json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(p, addr)
	return err
}

type DetectResult struct {
	LocalAddr           string
	PrimaryMappedAddr   string
	SecondaryMappedAddr string
}

type Detector struct {
	host      string
	primary   int
	secondary int
}

func NewDetector(host string, primary, secondary int) *Detector {
	return &Detector{
		host:      host,
		primary:   primary,
		secondary: secondary,
	}
}

func (d *Detector) Detect() (*DetectResult, error) {
	return Detect(d.host, d.primary, d.secondary)
}

func Detect(host string, primaryPort int, secondaryPort int) (*DetectResult, error) {
	var ip net.IP
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}
	if len(addrs) > 0 {
		host = addrs[0]
	}
	ip = net.ParseIP(host)
	if ip == nil {
		return nil, net.ErrClosed
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	s1, err := udpDetect(conn, ip, primaryPort)
	if err != nil {
		return nil, err
	}

	s2, err := udpDetect(conn, ip, secondaryPort)
	if err != nil {
		return nil, err
	}

	return &DetectResult{
		LocalAddr:           conn.LocalAddr().String(),
		PrimaryMappedAddr:   s1.MappedAddr,
		SecondaryMappedAddr: s2.MappedAddr,
	}, nil
}

func udpDetect(conn *net.UDPConn, ip net.IP, port int) (*mappedInfo, error) {
	p := make([]byte, 1)
	err := conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return nil, err
	}
	addr := &net.UDPAddr{IP: ip, Port: port}

	_, err = conn.WriteToUDP(p, addr)
	if err != nil {
		return nil, err
	}

	p = make([]byte, 1024)
	n, err := conn.Read(p)
	if err != nil {
		return nil, err
	}
	p = p[:n]
	resp := &mappedInfo{}
	err = json.Unmarshal(p, resp)
	return resp, err
}
