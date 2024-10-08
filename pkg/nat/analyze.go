package nat

import (
	"encoding/json"
	"net"
	"time"

	"github.com/elliotchance/pie/v2"
	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/pkg/network"
)

const (
	maxPortDistance = 256
)

type Action struct {
	Wait      time.Duration
	TryRemote bool
	LowTTL    bool
	Repeat    bool
}

type Resource struct {
	RemotePortStart int
	RemotePortEnd   int
	RemotePortCount int
	LocalPortCount  int
}

type AnalyzeResult struct {
	LocalAddrs        []string
	RemoteLocalAddrs  []string
	RemoteMappedAddrs []string
	Role              Role
	Resource          Resource
	Actions           []Action
}

func GetNatFromAnalyze(lresult *AnalyzeResult, rresult *AnalyzeResult) (localNat *Nat, remoteNat *Nat, err error) {
	localAddrs, err := network.ResolveUDPAddrs(lresult.LocalAddrs)
	if err != nil {
		return nil, nil, err
	}
	remoteAddrs, err := network.ResolveUDPAddrs(lresult.RemoteMappedAddrs)
	if err != nil {
		return nil, nil, err
	}
	rLocalAddrs, err := network.ResolveUDPAddrs(rresult.LocalAddrs)
	if err != nil {
		return nil, nil, err
	}
	rRemoteAddrs, err := network.ResolveUDPAddrs(rresult.RemoteMappedAddrs)
	if err != nil {
		return nil, nil, err
	}
	localNat = &Nat{
		LocalAddrs:        localAddrs,
		RemoteMappedAddrs: remoteAddrs,
		Role:              lresult.Role,
		Resource:          lresult.Resource,
		Actions:           lresult.Actions,
	}
	remoteNat = &Nat{
		LocalAddrs:        rLocalAddrs,
		RemoteMappedAddrs: rRemoteAddrs,
		Role:              rresult.Role,
		Resource:          rresult.Resource,
		Actions:           rresult.Actions,
	}
	return localNat, remoteNat, nil
}

func Analyze(local *DetectResult, remote *DetectResult) (lresult *AnalyzeResult, rresult *AnalyzeResult, err error) {
	hardLocal := isRandomPort(local) || isMultiExternIP(local)
	hardRemote := isRandomPort(remote) || isMultiExternIP(remote)

	switch {
	case !hardLocal && !hardRemote:
		lresult, rresult, err = analyzeDoubleEasy(local, remote)
	case hardLocal && !hardRemote:
		lresult, rresult, err = analyzeHasEasy(local, remote)
	case !hardLocal && hardRemote:
		rresult, lresult, err = analyzeHasEasy(remote, local)
	case hardLocal && hardRemote:
		lresult, rresult, err = analyzeDoubleHard(local, remote)
	}
	if err != nil {
		return nil, nil, err
	}

	a, _ := json.Marshal(lresult)
	b, _ := json.Marshal(rresult)
	logrus.Debugf("analyze result left = %s, right = %s", string(a), string(b))
	return lresult, rresult, nil
}

func analyzeDoubleEasy(local *DetectResult, remote *DetectResult) (lresult *AnalyzeResult, rresult *AnalyzeResult, err error) {
	lresult = &AnalyzeResult{
		LocalAddrs:        []string{local.LocalAddr},
		RemoteLocalAddrs:  filter([]string{remote.LocalAddr}),
		RemoteMappedAddrs: filter([]string{remote.PrimaryMappedAddr, remote.SecondaryMappedAddr}),
		Role:              ClientSide,
		Resource: Resource{
			LocalPortCount:  1,
			RemotePortCount: 1,
		},
		Actions: []Action{
			{
				Wait: 1 * time.Second,
			},
			{
				Repeat:    true,
				TryRemote: true,
			},
		},
	}
	rresult = &AnalyzeResult{
		LocalAddrs:        []string{remote.LocalAddr},
		RemoteLocalAddrs:  filter([]string{local.LocalAddr}),
		RemoteMappedAddrs: filter([]string{local.PrimaryMappedAddr, local.SecondaryMappedAddr}),
		Role:              ServerSide,
		Resource: Resource{
			LocalPortCount:  1,
			RemotePortCount: 1,
		},
		Actions: []Action{
			{
				TryRemote: true,
				LowTTL:    true,
			},
			{
				Repeat:    true,
				TryRemote: true,
				LowTTL:    true,
			},
		},
	}
	return lresult, rresult, nil
}

// analyzeHasEasy local is hard, remote is easy
func analyzeHasEasy(local *DetectResult, remote *DetectResult) (lresult *AnalyzeResult, rresult *AnalyzeResult, err error) {
	remoteAddrs, err := resolve([]string{
		remote.LocalAddr, remote.PrimaryMappedAddr, remote.SecondaryMappedAddr,
	})
	if err != nil {
		return nil, nil, err
	}
	lresult = &AnalyzeResult{
		LocalAddrs:        []string{local.LocalAddr},
		RemoteLocalAddrs:  filter([]string{remote.LocalAddr}),
		RemoteMappedAddrs: filter([]string{remote.PrimaryMappedAddr, remote.SecondaryMappedAddr}),
		Role:              ClientSide,
		Resource: Resource{
			LocalPortCount:  256,
			RemotePortCount: 1,
			RemotePortStart: remoteAddrs[1].Port,
		},
		Actions: []Action{
			{
				Wait: 1 * time.Second,
			},
			{
				Repeat:    true,
				TryRemote: true,
			},
		},
	}
	ls, le, _ := portsDistance(local)
	if ls > 10000 {
		ls = 10000
	}
	if le < 65000 {
		le = 65000
	}
	rresult = &AnalyzeResult{
		LocalAddrs:        []string{remote.LocalAddr},
		RemoteLocalAddrs:  filter([]string{local.LocalAddr}),
		RemoteMappedAddrs: filter([]string{local.PrimaryMappedAddr, local.SecondaryMappedAddr}),
		Role:              ServerSide,
		Resource: Resource{
			LocalPortCount:  1,
			RemotePortCount: 1024,
			RemotePortStart: ls,
			RemotePortEnd:   le,
		},
		Actions: []Action{
			{
				TryRemote: true,
				LowTTL:    true,
			},
			{
				Repeat:    true,
				TryRemote: true,
				LowTTL:    true,
			},
		},
	}
	return lresult, rresult, nil
}

func analyzeDoubleHard(local *DetectResult, remote *DetectResult) (lresult *AnalyzeResult, rresult *AnalyzeResult, err error) {
	// lps, lpe, lpd := portsDistance(local)
	// rps, rpe, rpd := portsDistance(remote)
	// if lpd > maxPortDistance && rpd > maxPortDistance {
	// 	return nil, nil, errHardToSuccess
	// }
	lresult = &AnalyzeResult{
		LocalAddrs:        []string{local.LocalAddr},
		RemoteMappedAddrs: filter([]string{remote.LocalAddr, remote.PrimaryMappedAddr, remote.SecondaryMappedAddr}),
		Role:              ClientSide,
		Resource: Resource{
			LocalPortCount:  256,
			RemotePortCount: 1024,
			RemotePortStart: 10000,
			RemotePortEnd:   65000,
		},
		Actions: []Action{
			{
				TryRemote: true,
				LowTTL:    true,
			},
			{
				TryRemote: true,
				Repeat:    true,
			},
		},
	}
	ls, le, _ := portsDistance(local)
	if ls > 10000 {
		ls = 10000
	}
	if le < 65000 {
		le = 65000
	}
	rresult = &AnalyzeResult{
		LocalAddrs:        []string{remote.LocalAddr},
		RemoteMappedAddrs: filter([]string{local.LocalAddr, local.PrimaryMappedAddr, local.SecondaryMappedAddr}),
		Role:              ServerSide,
		Resource: Resource{
			LocalPortCount:  256,
			RemotePortCount: 1024,
			RemotePortStart: ls,
			RemotePortEnd:   le,
		},
		Actions: []Action{
			{
				TryRemote: true,
				LowTTL:    true,
			},
			{
				TryRemote: true,
				Repeat:    true,
			},
		},
	}
	return lresult, rresult, nil
}

func filter(addrs []string) []string {
	amap := make(map[string]bool)
	for _, a := range addrs {
		amap[a] = true
	}
	return pie.Keys(amap)
}

func resolve(addrs []string) ([]*net.UDPAddr, error) {
	results := make([]*net.UDPAddr, 0)
	for _, addr := range addrs {
		r, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func isMultiExternIP(r *DetectResult) bool {
	primary, _ := net.ResolveUDPAddr("udp", r.PrimaryMappedAddr)
	secondary, _ := net.ResolveUDPAddr("udp", r.SecondaryMappedAddr)
	return primary.IP.String() != secondary.IP.String()
}

func isRandomPort(r *DetectResult) bool {
	primary, _ := net.ResolveUDPAddr("udp", r.PrimaryMappedAddr)
	secondary, _ := net.ResolveUDPAddr("udp", r.SecondaryMappedAddr)
	return primary.Port != secondary.Port
}

func portsDistance(r *DetectResult) (int, int, int) {
	primary, _ := net.ResolveUDPAddr("udp", r.PrimaryMappedAddr)
	secondary, _ := net.ResolveUDPAddr("udp", r.SecondaryMappedAddr)
	if secondary.Port-primary.Port > 0 {
		return primary.Port, secondary.Port, secondary.Port - primary.Port
	}
	return secondary.Port, primary.Port, primary.Port - secondary.Port
}
