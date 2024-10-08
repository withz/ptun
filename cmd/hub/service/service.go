package service

import (
	"context"

	"github.com/withz/ptun/app/config"
	"github.com/withz/ptun/pkg/hub"
	"github.com/withz/ptun/pkg/nat"
)

type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Start(ctx context.Context) error {
	go s.Run(ctx)
	return nil
}

func (s *Service) Run(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	natServer := nat.NewSimpleServer(config.Server().Stun.PrimaryPort, config.Server().Stun.SecondaryPort)
	err := natServer.Start()
	if err != nil {
		return err
	}

	h := hub.NewHub(
		hub.NewTcpHubServer(&hub.TcpHubServerConfig{
			Port:  config.Server().ServerPort,
			Token: config.Server().Token,
		}),
	)
	err = h.Start()
	if err != nil {
		return err
	}

	<-s.ctx.Done()
	h.Close()
	natServer.Stop()
	return nil
}

func (s *Service) Close() {
	s.cancel()
}
