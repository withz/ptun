package hub

import (
	"reflect"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/model"
	"github.com/withz/ptun/pkg/nat"
	"github.com/withz/ptun/pkg/proto"
)

type HubServer interface {
	Start() error
	Close() error
	Accept() <-chan *session
}

type Hub struct {
	sessions sync.Map
	servers  []HubServer
}

func NewHub(servers ...HubServer) *Hub {
	return &Hub{
		servers: servers,
	}
}

func (h *Hub) Start() error {
	for _, s := range h.servers {
		err := s.Start()
		if err != nil {
			return err
		}
		go func() {
			for {
				session, ok := <-s.Accept()
				if !ok {
					return
				}
				go h.handle(session)
			}
		}()
	}
	return nil
}

func (h *Hub) Close() error {
	for _, s := range h.servers {
		s.Close()
	}
	return nil
}

func (h *Hub) saveSession(s *session) {
	old := h.loadSession(s.name)
	if old != nil {
		old.Close()
	}
	h.sessions.Store(s.name, s)
}

func (h *Hub) removeSession(name string) {
	old := h.loadSession(name)
	if old != nil {
		old.Close()
	}
	h.sessions.Delete(name)
}

func (h *Hub) loadSession(name string) *session {
	s, ok := h.sessions.Load(name)
	if !ok {
		return nil
	}
	return s.(*session)
}

func (h *Hub) allSessionNames() (names []string) {
	names = make([]string, 0)
	h.sessions.Range(func(key, value any) bool {
		names = append(names, key.(string))
		return true
	})
	return names
}

func (h *Hub) handle(session *session) {
	logrus.Debugf("new seesion come %s", session.name)
	handler := NewHubHandler(session, h)
	dispatcher := handler.session.Requester.Dispatcher()
	dispatcher.AddHandler(reflect.TypeFor[model.PeerListRequest]().Name(), handler.handlePeerList)
	dispatcher.AddHandler(reflect.TypeFor[model.PunchRequest]().Name(), handler.handlePunch)
	h.saveSession(session)
	handler.session.RunDispatcher()
	h.removeSession(session.name)
	handler.session.Close()
	logrus.Debugf("seesion leave %s", session.name)
}

type hubHandler struct {
	session *session
	hub     *Hub
}

func NewHubHandler(session *session, hub *Hub) *hubHandler {
	return &hubHandler{
		session: session,
		hub:     hub,
	}
}

func (h *hubHandler) handlePeerList(r *proto.Request) {
	logrus.Debugf("[%s] recv peer list request", h.session.name)
	h.session.Responser.ReplySuccess(r, &model.PeerListResponse{
		PeerNames: h.hub.allSessionNames(),
	})
}

func (h *hubHandler) handlePunch(r *proto.Request) {
	logrus.Debugf("[%s] recv punch request, %v", h.session.name, r.Payload())
	req, err := proto.GetPayload[model.PunchRequest](r)
	if err != nil {
		// todo
		return
	}

	remoteSession := h.hub.loadSession(req.PeerName)
	if remoteSession == nil {
		logrus.Debugf("peer %s not found in sessions", req.PeerName)
		// todo
		return
	}
	resp, err := remoteSession.SendMessage(&model.DetectNatRequest{}, 3*time.Second)
	if err != nil {
		// todo
		logrus.Debugf("peer %s send failed", req.PeerName)
		return
	}
	detectResult, err := proto.GetPayload[model.DetectNatResponse](resp)
	if err != nil {
		// todo
		logrus.Debugf("peer %s do not give response", req.PeerName)
		return
	}
	remoteIp := detectResult.Ip
	remote := detectResult.Local.Mapping
	local := req.Local.Mapping

	lr, rr, err := nat.Analyze(&local, &remote)
	if err != nil {
		// todo
		logrus.Debugf("nat analyze failed")
		return
	}
	h.session.Responser.SendSuccess(&model.PunchResponse{
		LocalIp:        req.LocalIp,
		RemoteIp:       remoteIp,
		LocalNat:       *lr,
		RemoteNat:      *rr,
		RemotePeerName: remoteSession.name,
	})
	remoteSession.Responser.SendSuccess(&model.PunchResponse{
		LocalIp:        remoteIp,
		RemoteIp:       req.LocalIp,
		LocalNat:       *rr,
		RemoteNat:      *lr,
		RemotePeerName: h.session.name,
	})
}
