package neigh

import (
	"net"

	"github.com/vishvananda/netlink"
)

const (
	NUD_REACHABLE = 0x02
)

func SetNeighOverIface(ifIndex int, eip net.IP, macAddr net.HardwareAddr) {
	neigh := &netlink.Neigh{
		LinkIndex:    ifIndex,
		State:        NUD_REACHABLE,
		IP:           eip,
		HardwareAddr: macAddr,
	}
	if err := netlink.NeighSet(neigh); err != nil {
		log.Warn().Err(err).Msgf(`fail to set neigh, ip: %s mac: %s`, eip.String(), macAddr.String())
		if err = netlink.NeighAdd(neigh); err != nil {
			log.Warn().Err(err).Msgf(`fail to add neigh, ip: %s mac: %s`, eip.String(), macAddr.String())
		}
	}
}

func DelNeighOverIface(ifIndex int, eip net.IP, macAddr net.HardwareAddr) {
	neigh := &netlink.Neigh{
		LinkIndex:    ifIndex,
		State:        NUD_REACHABLE,
		IP:           eip,
		HardwareAddr: macAddr,
	}
	if err := netlink.NeighDel(neigh); err != nil {
		log.Warn().Err(err).Msgf(`fail to del neigh, ip: %s mac: %s`, eip.String(), macAddr.String())
	}
}
