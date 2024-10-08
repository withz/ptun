package model

import "net"

type UpdateIP struct {
	IPs []*net.IPNet
}

type UpdateRoute struct {
	Routes []*net.IPNet
}
