package app

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/withz/ptun/pkg/hub"
	"github.com/withz/ptun/pkg/nat"
)

type NodeConfig struct {
	StunHost    string
	StunPriPort int
	StunSecPort int
	HubHost     string
	HubPort     int
	HubToken    string
	NodeIP      string
}

type Node struct {
	cfg  *NodeConfig
	name string
}

func CreateNode(cfg *NodeConfig) *Node {
	return &Node{
		cfg: cfg,
	}
}

func (n *Node) Run(nw *P2PNetwork) error {
	detector := nat.NewDetector(n.cfg.StunHost, n.cfg.StunPriPort, n.cfg.StunSecPort)
	for {
		ex, err := hub.NewExchanger(hub.NewTcpHubClient(&hub.TcpHubClientConfig{
			Host:       n.cfg.HubHost,
			Port:       n.cfg.HubPort,
			ClientName: n.name,
			Token:      n.cfg.HubToken,
		}), detector, n.cfg.NodeIP)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		n.name = ex.GetName()
		go func() {
			for m := range ex.Accept() {
				err := nw.newNatPeer(m.PeerName, m.PeerIP, n.cfg.HubToken, m.NatMessage)
				if err != nil {
					logrus.Infof("new nat peer err, %s", err.Error())
				}
			}
		}()
		for {
			peers, err := ex.GetPeers()
			if err != nil {
				break
			}
			for _, p := range peers {
				if p == n.name || nw.hasPeer(p) {
					continue
				}
				ex.PunchPeer(p, n.cfg.NodeIP)
			}
			time.Sleep(5 * time.Second)
		}
	}
}
