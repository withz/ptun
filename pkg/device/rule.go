package device

import (
	"net"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
)

type RuleManager struct {
	iptables *iptables.IPTables
	rules    map[string][]string
}

func NewRuleManager() (*RuleManager, error) {
	ipt, err := iptables.New()
	if err != nil {
		return nil, err
	}
	return &RuleManager{
		iptables: ipt,
		rules:    make(map[string][]string),
	}, nil
}

func (rm *RuleManager) UpdateIptables(peerNet string, dsts []*net.IPNet) error {
	table := "nat"
	chain := "POSTROUTING"
	updated := make(map[string]bool)
	for _, dst := range dsts {
		rule := []string{"-s", peerNet, "-d", dst.String(), "-j", "MASQUERADE"}
		exists, err := rm.iptables.Exists(table, chain, rule...)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		err = rm.iptables.Append(table, chain, rule...)
		if err != nil {
			return err
		}
		rm.rules[dst.String()] = rule
		updated[dst.String()] = true
	}
	for src, rule := range rm.rules {
		if _, ok := updated[src]; ok {
			continue
		}
		if _, ok := rm.rules[src]; !ok {
			continue
		}
		exists, err := rm.iptables.Exists(table, chain, rule...)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		err = rm.iptables.Delete(table, chain, rule...)
		if err != nil {
			return err
		}
		delete(rm.rules, src)
	}

	logrus.Debugf("udpate iptables, %v", rm.rules)
	return nil
}

func (rm *RuleManager) ClearAllRules() {
	table := "nat"
	chain := "POSTROUTING"
	for src, rule := range rm.rules {
		exists, err := rm.iptables.Exists(table, chain, rule...)
		if err != nil {
			continue
		}
		if !exists {
			continue
		}
		err = rm.iptables.Delete(table, chain, rule...)
		if err != nil {
			continue
		}
		delete(rm.rules, src)
	}

	logrus.Debugf("clear iptables")
}
