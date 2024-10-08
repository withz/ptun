package bridge

import (
	"net"

	"github.com/withz/ptun/pkg/proto"
)

type Relay struct {
	*proto.Transport
	name   string
	routes []*net.IPNet
}
