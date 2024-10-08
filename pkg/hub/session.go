package hub

import (
	"github.com/withz/ptun/pkg/proto"
)

type session struct {
	*proto.Transport
	name string
}

func NewSession(name string, conn *proto.Transport) *session {
	return &session{
		Transport: conn,
		name:      name,
	}
}
