// Code generated .* DO NOT EDIT

package model

import (
	"reflect"

	"github.com/withz/ptun/pkg/proto"
)

func init() {
	proto.RegisterMessage(reflect.TypeFor[LoginRequest]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[LoginResponse]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[PeerListRequest]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[PeerListResponse]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[PeerNatInfo]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[ExchangeNatRequest]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[ExchangeNatResponse]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[DetectNatRequest]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[DetectNatResponse]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[PunchRequest]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[PunchResponse]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[UpdateIP]())
}

func init() {
	proto.RegisterMessage(reflect.TypeFor[UpdateRoute]())
}
