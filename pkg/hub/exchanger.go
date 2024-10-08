package hub

import (
	"reflect"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/model"
	"github.com/withz/ptun/pkg/nat"
	"github.com/withz/ptun/pkg/network"
	"github.com/withz/ptun/pkg/proto"
)

const (
	LoginConnectionTimeout = 10 * time.Second
	LoginRepeatWaitTime    = 10 * time.Second
	LoginRepeatCount       = 3
)

type Exchanger struct {
	session  *session
	detector *nat.Detector
	ip       string
	info     chan *ExchangeInfo
}

func NewExchanger(c HubClient, d *nat.Detector, ip string) (*Exchanger, error) {
	s, err := tryLogin(c)
	if err != nil {
		return nil, err
	}
	e := &Exchanger{
		session:  s,
		detector: d,
		info:     make(chan *ExchangeInfo),
		ip:       ip,
	}
	reqDispatcher := s.Requester.Dispatcher()
	reqDispatcher.AddHandler(reflect.TypeFor[model.DetectNatRequest]().Name(), e.handleDetectNat)
	go s.Requester.RunDispatcher()

	respDispatcher := s.Responser.Dispatcher()
	respDispatcher.AddHandler(reflect.TypeFor[model.PunchResponse]().Name(), e.handlePunch)
	go s.Responser.RunDispatcher()
	return e, nil
}

func (e *Exchanger) Close() error {
	return e.session.Close()
}

func (e *Exchanger) GetName() string {
	return e.session.name
}

func (e *Exchanger) GetPeers() ([]string, error) {
	raw, err := e.session.SendMessage(&model.PeerListRequest{}, LoginConnectionTimeout)
	if err != nil {
		return nil, err
	}
	resp, err := proto.GetResponsePayload[model.PeerListResponse](raw)
	if err != nil {
		return nil, err
	}
	return resp.PeerNames, nil
}

func (e *Exchanger) PunchPeer(name string, localIP string) (err error) {
	m, err := e.detector.Detect()
	if err != nil {
		return err
	}
	return e.session.Requester.Send(&model.PunchRequest{
		PeerName: name,
		LocalIp:  localIP,
		Local: model.PeerNatInfo{
			Name:    e.session.name,
			Mapping: *m,
		},
	})
}

func (e *Exchanger) Accept() <-chan *ExchangeInfo {
	return e.info
}

func (e *Exchanger) handleDetectNat(r *proto.Request) {
	n, err := e.detector.Detect()
	if err != nil {
		// todo return fmt.Errorf("detect nat failed, %w", err)
		logrus.Debug("detect nat failed")
	}
	e.session.Responser.ReplySuccess(r, &model.DetectNatResponse{
		Local: model.PeerNatInfo{
			Name:    e.session.name,
			Mapping: *n,
		},
		Ip: e.ip,
	})
}

type ExchangeInfo struct {
	NatMessage *nat.Nat
	PeerName   string
	PeerIP     string
}

func (e *Exchanger) handlePunch(r *proto.Response) {
	resp, err := proto.GetResponsePayload[model.PunchResponse](r)
	if err != nil {
		// todo return fmt.Errorf("parse response err, %w", err)
	}
	localAddrs, err := network.ResolveUDPAddrs(resp.LocalNat.LocalAddrs)
	if err != nil {
		logrus.Debugf("get err %s", err.Error())
		// todo return fmt.Errorf("parse local addrs err, %w", err)
	}
	remoteAddrs, err := network.ResolveUDPAddrs(resp.LocalNat.RemoteMappedAddrs)
	if err != nil {
		// todo return fmt.Errorf("parse remote addrs err, %w", err)
	}
	localNat := &nat.Nat{
		LocalAddrs:        localAddrs,
		RemoteMappedAddrs: remoteAddrs,
		Role:              resp.LocalNat.Role,
		Resource:          resp.LocalNat.Resource,
		Actions:           resp.LocalNat.Actions,
	}
	select {
	case e.info <- &ExchangeInfo{
		NatMessage: localNat,
		PeerName:   resp.RemotePeerName,
		PeerIP:     resp.RemoteIp,
	}:
	case <-e.session.Done():
	}
}

type HubClient interface {
	Login() (*session, error)
}

func tryLogin(c HubClient) (s *session, err error) {
	retryFunc := func() error {
		s, err = c.Login()
		if err != nil {
			logrus.Error(err.Error())
		}
		return err
	}
	err = backoff.Retry(
		retryFunc,
		backoff.WithMaxRetries(
			backoff.NewConstantBackOff(LoginRepeatWaitTime),
			LoginRepeatCount,
		),
	)
	return s, err
}
