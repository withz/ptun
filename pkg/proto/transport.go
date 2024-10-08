package proto

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/pkg/tools"
)

const maxAliveCount = 5

type requester struct {
	transport  *Transport
	recvCh     chan *Packet
	dispatcher *Dispatcher[*Request]
}

func (r *requester) Send(data any) error {
	return r.Write(NewRequest(data))
}

func (r *requester) Write(req *Request) error {
	p, err := req.Pack()
	if err != nil {
		return err
	}
	return PackInto(Req, p, r.transport.conn)
}

func (r *requester) Read(d time.Duration) (*Request, error) {
	select {
	case p := <-r.recvCh:
		defer p.Release()
		return UnpackRequest(p.body)
	case <-time.After(d):
		return nil, fmt.Errorf("read request timeout")
	}
}

func (r *requester) Dispatcher() *Dispatcher[*Request] {
	return r.dispatcher
}

func (r *requester) RunDispatcher() {
	for {
		p, ok := <-r.recvCh
		if !ok {
			return
		}
		defer p.Release()
		req, err := UnpackRequest(p.body)
		if err != nil {
			continue
		}
		err = r.dispatcher.Dispatch(req)
		if err != nil {
			continue
		}
	}
}

type responser struct {
	transport  *Transport
	recvCh     chan *Packet
	dispatcher *Dispatcher[*Response]
	replyer    map[uint32]chan *Response
}

func (r *responser) ReplySuccess(req *Request, data any) error {
	return r.SendSuccessWithID(req.Id, data)
}

func (r *responser) Reply(req *Request, code int, message string, data any) error {
	return r.SendWithID(req.Id, code, message, data)
}

func (r *responser) SendSuccess(data any) error {
	return r.SendSuccessWithID(Next(), data)
}

func (r *responser) SendSuccessWithID(id uint32, data any) error {
	return r.SendWithID(id, 0, "success", data)
}

func (r *responser) Send(code int, message string, data any) error {
	return r.SendWithID(Next(), code, message, data)
}

func (r *responser) SendWithID(id uint32, code int, message string, data any) error {
	return r.Write(NewIdResponse(id, code, message, data))
}

func (r *responser) Write(resp *Response) error {
	p, err := resp.Pack()
	if err != nil {
		return err
	}
	return PackInto(Resp, p, r.transport.conn)
}

func (r *responser) Read(d time.Duration) (*Response, error) {
	select {
	case p := <-r.recvCh:
		if p == nil {
			return nil, fmt.Errorf("read response timeout")
		}
		defer p.Release()
		return UnpackResponse(p.body)
	case <-time.After(d):
		return nil, fmt.Errorf("read response timeout")
	}
}

func (r *responser) Dispatcher() *Dispatcher[*Response] {
	return r.dispatcher
}

func (r *responser) RunDispatcher() {
	for {
		p, ok := <-r.recvCh
		if !ok {
			return
		}
		defer p.Release()
		resp, err := UnpackResponse(p.body)
		if err != nil {
			continue
		}
		if ch, ok := r.replyer[resp.Id]; ok {
			ch <- resp
			continue
		}
		err = r.dispatcher.Dispatch(resp)
		if err != nil {
			continue
		}
	}
}

type Transport struct {
	Requester requester
	Responser responser

	conn  net.Conn
	rawCh chan *Packet

	aliveCount    int64
	aliveInterval time.Duration
	aliveInterupt chan struct{}
	closeOnce     sync.Once
	done          chan struct{}
}

func NewTransport(c net.Conn) *Transport {
	t := &Transport{
		conn:       c,
		rawCh:      make(chan *Packet, 100),
		aliveCount: maxAliveCount,
		done:       make(chan struct{}),
	}
	t.Requester = requester{
		transport:  t,
		recvCh:     make(chan *Packet, 10),
		dispatcher: NewDispatcher[*Request](),
	}
	t.Responser = responser{
		transport:  t,
		recvCh:     make(chan *Packet, 10),
		dispatcher: NewDispatcher[*Response](),
		replyer:    map[uint32]chan *Response{},
	}
	go t.readloop()
	return t
}

func (t *Transport) SetKeepalive(v time.Duration) {
	if v == t.aliveInterval {
		return
	}
	if t.aliveInterupt != nil {
		close(t.aliveInterupt)
		t.aliveInterupt = nil
	}
	if v == 0 {
		return
	}
	t.aliveInterupt = make(chan struct{})
	atomic.StoreInt64(&t.aliveCount, maxAliveCount)

	isDone := func(i time.Duration, interupter chan struct{}) bool {
		select {
		case <-t.done:
			return true
		case <-interupter:
			return true
		case <-time.After(i):
			return false
		}
	}
	go func(interupter chan struct{}) {
		for !isDone(t.aliveInterval/time.Duration(maxAliveCount), interupter) {
			atomic.AddInt64(&t.aliveCount, -1)
			PackInto(Ping, nil, t.conn)
			time.Sleep(1 * time.Second)
		}
	}(t.aliveInterupt)
	go func(interupter chan struct{}) {
		for !isDone(1*time.Second, interupter) {
			if t.aliveCount < 0 {
				_ = t.Close()
			}
		}
	}(t.aliveInterupt)
}

func (t *Transport) SendMessage(data any, timeout time.Duration) (*Response, error) {
	req := NewRequest(data)
	err := t.Requester.Write(req)
	if err != nil {
		return nil, err
	}
	respCh := make(chan *Response)
	t.Responser.replyer[req.Id] = respCh
	defer func() {
		delete(t.Responser.replyer, req.Id)
		close(respCh)
	}()
	select {
	case resp, ok := <-t.Responser.replyer[req.Id]:
		if !ok {
			return nil, fmt.Errorf("send message timeout")
		}
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("send message timeout")
	}
}

func (t *Transport) RunDispatcher() {
	for {
		select {
		case p, ok := <-t.Requester.recvCh:
			if !ok {
				logrus.Debugf("request dispatcher exit")
				return
			}
			defer p.Release()
			req, err := UnpackRequest(p.body)
			if err != nil {
				continue
			}
			err = t.Requester.dispatcher.Dispatch(req)
			if err != nil {
				continue
			}
		case p, ok := <-t.Responser.recvCh:
			if !ok {
				logrus.Debugf("response dispatcher exit")
				return
			}
			defer p.Release()
			resp, err := UnpackResponse(p.body)
			if err != nil {
				continue
			}
			if ch, ok := t.Responser.replyer[resp.Id]; ok {
				ch <- resp
				continue
			}
			err = t.Responser.dispatcher.Dispatch(resp)
			if err != nil {
				continue
			}
		}
	}
}

func (t *Transport) readloop() {
	defer func() {
		logrus.Debugf("transport readloop exit")
	}()
	reader := bufio.NewReader(t.conn)
	for {
		pkt, err := UnpackFrom(reader)
		if err == io.EOF || tools.IsNetError(err) {
			logrus.Debugf("transport readloop err, %s", err.Error())
			t.Close()
			return
		}
		if err != nil {
			continue
		}
		atomic.StoreInt64(&t.aliveCount, maxAliveCount)
		select {
		case <-t.done:
			return
		default:
		}
		switch pkt.Tag() {
		case Raw:
			select {
			case <-t.done:
				return
			case t.rawCh <- pkt:
			}
		case Req:
			if pkt == nil || pkt.body == nil {
				continue
			}
			select {
			case <-t.done:
				return
			case t.Requester.recvCh <- pkt:
			}
		case Resp:
			if pkt == nil || pkt.body == nil {
				continue
			}
			select {
			case <-t.done:
				return
			case t.Responser.recvCh <- pkt:
			}
		case Ping:
			PackInto(Pong, nil, t.conn)
		case Pong:
		}
	}
}

func (t *Transport) Done() <-chan struct{} {
	return t.done
}

func (t *Transport) Read(b []byte) (n int, err error) {
	p, ok := <-t.rawCh
	if !ok {
		return 0, fmt.Errorf("transport read err, channel closed")
	}
	defer p.Release()
	if len(b) < len(p.body) {
		return 0, fmt.Errorf("transport read err, buf length too short")
	}
	copy(b, p.body[:])
	return len(p.body), nil
}

func (t *Transport) Write(b []byte) (n int, err error) {
	err = PackInto(Raw, b, t.conn)
	if err != nil {
		err = fmt.Errorf("transport write err, %w", err)
	}
	return len(b), err
}

func (t *Transport) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)
		close(t.rawCh)
		close(t.Requester.recvCh)
		close(t.Responser.recvCh)
	})
	return t.conn.Close()
}

func (t *Transport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *Transport) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

func (t *Transport) SetDeadline(v time.Time) error {
	return t.conn.SetDeadline(v)
}

func (t *Transport) SetReadDeadline(v time.Time) error {
	return t.conn.SetReadDeadline(v)
}

func (t *Transport) SetWriteDeadline(v time.Time) error {
	return t.conn.SetWriteDeadline(v)
}
