package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/withz/ptun/app"
	"github.com/withz/ptun/app/config"
	"github.com/withz/ptun/pkg/hub"
	"github.com/withz/ptun/pkg/nat"
)

type Service struct {
	clientName string

	network *app.P2PNetwork
	ctx     context.Context
	cancel  context.CancelFunc
}

const (
	LoginRepeatWaitTime    = 10 * time.Second
	LoginRepeatCount       = 3
	LoginConnectionTimeout = 10 * time.Second
)

func NewService() *Service {
	return &Service{}
}

func (s *Service) Start(ctx context.Context) (err error) {
	cfg := config.Client().Net
	s.network, err = app.CreateNet(&app.P2PNetworkConfig{
		Tun:       cfg.Tun,
		IP:        cfg.IP,
		AllowNets: cfg.AllowNets,
		Routers:   cfg.Routers,
	})
	if err != nil {
		return err
	}
	go s.Run(ctx)
	return nil
}

func (s *Service) Run(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	stun := config.Client().Stun
	detector := nat.NewDetector(stun.Host, stun.PrimaryPort, stun.SecondaryPort)

	for {
		ex, err := hub.NewExchanger(hub.NewTcpHubClient(&hub.TcpHubClientConfig{
			Host:       config.Client().ServerHost,
			Port:       config.Client().ServerPort,
			ClientName: s.clientName,
			Token:      config.Client().Token,
		}), detector, config.Client().Net.IP)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		s.clientName = ex.GetName()
		go func() {
			for m := range ex.Accept() {
				logrus.Debugf("peer %s, ip = %s come", m.PeerName, m.PeerIP)
				err := s.network.NewNatPeer(m.PeerName, m.PeerIP, config.Client().Token, m.NatMessage)
				if err != nil {
					logrus.Infof("new nat peer err, %s", err.Error())
				}
			}
		}()
		for {
			peers, err := ex.GetPeers()
			if err != nil {
				break
			}
			for _, p := range peers {
				if p == s.clientName || s.network.HasPeer(p) {
					continue
				}
				ex.PunchPeer(p, config.Client().Net.IP)
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func (s *Service) Close() {
	s.cancel()
	if s.network != nil {
		s.network.OnShutdown()
	}
}
