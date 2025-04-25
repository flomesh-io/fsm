package neigh

import (
	"net"

	utilnet "k8s.io/utils/net"
)

// GratuitousNeighOverIface sends a gratuitous packet over interface 'iface' from 'eip'.
func GratuitousNeighOverIface(ifName string, ifIndex int, eip net.IP, macAddr net.HardwareAddr) {
	if utilnet.IsIPv6(eip) {
		if err := gratuitousNDPOverIface(eip.To16(), ifName, macAddr); err != nil {
			log.Error().Err(err).Msgf(`fail to gratuitous ndp over iface, ifIndex: %d ip: %s mac: %s`, ifIndex, eip.String(), macAddr.String())
		}
	}
	if utilnet.IsIPv4(eip) {
		if err := gratuitousARPOverIface(eip.To4(), ifName, ifIndex, macAddr); err != nil {
			log.Error().Err(err).Msgf(`fail to gratuitous arp over iface, ifIndex: %d ip: %s mac: %s`, ifIndex, eip.String(), macAddr.String())
		}
	}
}
