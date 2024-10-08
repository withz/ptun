package device

import (
	"net"

	"github.com/vishvananda/netlink"
)

type tunIface struct {
	name string
}

func (t *tunIface) AddAddr(addrs ...*net.IPNet) error {
	iface, err := netlink.LinkByName(t.name)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		err = netlink.AddrAdd(iface, &netlink.Addr{
			IPNet: addr,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tunIface) DelAddr(addrs ...*net.IPNet) error {
	iface, err := netlink.LinkByName(t.name)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		err = netlink.AddrDel(iface, &netlink.Addr{
			IPNet: addr,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (t *tunIface) AddRoute(routes ...*net.IPNet) error {
	iface, err := netlink.LinkByName(t.name)
	if err != nil {
		return err
	}
	for _, route := range routes {
		err := netlink.RouteAdd(&netlink.Route{
			Dst:       route,
			Gw:        net.IPv4(0, 0, 0, 0),
			LinkIndex: iface.Attrs().Index,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tunIface) DelRoute(routes ...*net.IPNet) error {
	iface, err := netlink.LinkByName(t.name)
	if err != nil {
		return err
	}
	for _, route := range routes {
		err := netlink.RouteDel(&netlink.Route{
			Dst:       route,
			Gw:        net.IPv4(0, 0, 0, 0),
			LinkIndex: iface.Attrs().Index,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (t *tunIface) Up() error {
	iface, err := netlink.LinkByName(t.name)
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(iface)
}
func (t *tunIface) Down() error {
	iface, err := netlink.LinkByName(t.name)
	if err != nil {
		return err
	}
	return netlink.LinkSetDown(iface)
}
