package hub

import (
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/model"
	"github.com/withz/ptun/pkg/proto"
	"github.com/withz/ptun/pkg/tools"
)

type TcpHubServerConfig struct {
	Port  int
	Token string
}

type TcpHubServer struct {
	cfg       *TcpHubServerConfig
	listener  net.Listener
	sessionCh chan *session
}

func NewTcpHubServer(cfg *TcpHubServerConfig) *TcpHubServer {
	if cfg == nil {
		panic("config cannot be nil")
	}
	return &TcpHubServer{
		cfg:       cfg,
		sessionCh: make(chan *session),
	}
}

func (s *TcpHubServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return err
	}
	s.listener = listener
	go func() {
		defer func() {
			logrus.Debugf("tcp server accept loop exit")
		}()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go s.handleLogin(conn)
		}
	}()
	return nil
}

func (s *TcpHubServer) Close() error {
	if s.listener != nil {
		s.listener.Close()
	}
	close(s.sessionCh)
	return nil
}

func (s *TcpHubServer) Accept() <-chan *session {
	return s.sessionCh
}

func (s *TcpHubServer) handleLogin(conn net.Conn) {
	t := proto.NewTransport(conn)
	req, err := t.Requester.Read(5 * time.Second)
	if err != nil {
		logrus.Infof("wait for login failed, %s", err.Error())
		t.Close()
		return
	}
	login, err := proto.GetPayload[model.LoginRequest](req)
	if err != nil {
		logrus.Infof("login failed, %s", err.Error())
		t.Close()
		return
	}
	if login.Token != s.cfg.Token {
		logrus.Infof("login failed, invalid token")
		t.Close()
		return
	}
	if login.Name == "" {
		login.Name = tools.GenUUID()
	}
	err = t.Responser.SendSuccess(&model.LoginResponse{
		Name: login.Name,
	})
	if err != nil {
		logrus.Infof("login failed, %s", err.Error())
		t.Close()
		return
	}
	session := NewSession(login.Name, t)
	s.sessionCh <- session
}

type TcpHubClientConfig struct {
	Host       string
	Port       int
	ClientName string
	Token      string
}

type TcpHubClient struct {
	cfg *TcpHubClientConfig
}

func NewTcpHubClient(cfg *TcpHubClientConfig) *TcpHubClient {
	if cfg == nil {
		panic("config cannot be nil")
	}
	return &TcpHubClient{
		cfg: cfg,
	}
}

func (c *TcpHubClient) Login() (*session, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	t := proto.NewTransport(conn)

	t.SetDeadline(time.Now().Add(LoginConnectionTimeout))
	defer t.SetDeadline(time.Time{})

	err = t.Requester.Send(&model.LoginRequest{
		Name:  c.cfg.ClientName,
		Token: c.cfg.Token,
	})
	if err != nil {
		return nil, fmt.Errorf("login failed, %w", err)
	}
	resp, err := t.Responser.Read(LoginConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("login failed, %w", err)
	}
	login, err := proto.GetPayload[model.LoginResponse](resp)
	if err != nil {
		return nil, fmt.Errorf("login failed, %w", err)
	}
	session := NewSession(login.Name, t)
	return session, nil
}
