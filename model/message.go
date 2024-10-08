package model

import "github.com/withz/ptun/pkg/nat"

type LoginRequest struct {
	Name  string
	Token string
}

type LoginResponse struct {
	Name         string
	ConnectionId string
}

type PeerListRequest struct {
}

type PeerListResponse struct {
	PeerNames []string
}

type PeerNatInfo struct {
	Name    string
	Mapping nat.DetectResult
}

type ExchangeNatRequest struct {
	Local    PeerNatInfo
	PeerName string
}

type ExchangeNatResponse struct {
	Remote PeerNatInfo
}

type DetectNatRequest struct {
	Remote *PeerNatInfo `json:"Remote,omitempty"`
}

type DetectNatResponse struct {
	Ip    string
	Local PeerNatInfo
}

type PunchRequest struct {
	Local    PeerNatInfo
	LocalIp  string
	PeerName string
}

type PunchResponse struct {
	LocalIp        string
	RemoteIp       string
	LocalNat       nat.AnalyzeResult
	RemoteNat      nat.AnalyzeResult
	RemotePeerName string
}
