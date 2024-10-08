package proto

import (
	"fmt"
)

type (
	Message interface {
		GetKey() string
		Payload() any
	}

	Handler[T Message] func(payload T)

	Dispatcher[T Message] struct {
		handlers map[string]Handler[T]
	}
)

func NewDispatcher[T Message]() *Dispatcher[T] {
	d := &Dispatcher[T]{
		handlers: map[string]Handler[T]{},
	}
	return d
}

func (d *Dispatcher[T]) AddHandler(r string, h Handler[T]) {
	d.handlers[r] = h
}

func (d *Dispatcher[T]) Dispatch(r T) error {
	h, ok := d.handlers[r.GetKey()]
	if !ok {
		return fmt.Errorf("no handler of %s", r.GetKey())
	}
	go h(r)
	return nil
}

func GetPayload[V any](e Message) (*V, error) {
	if e.Payload() == nil {
		return nil, fmt.Errorf("message has no payload")
	}
	if err := CheckType[V](e.Payload()); err != nil {
		return nil, err
	}
	result := e.Payload().(*V)
	if result == nil {
		return nil, fmt.Errorf("invalid message")
	}
	return result, nil
}
