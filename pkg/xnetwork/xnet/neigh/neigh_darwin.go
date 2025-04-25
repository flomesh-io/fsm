package neigh

import (
	"net"
)

// GratuitousNeighOverIface sends a gratuitous packet over interface 'iface' from 'eip'.
func GratuitousNeighOverIface(ifName string, ifIndex int, eip net.IP, macAddr net.HardwareAddr) {
	panic("Unsupported!")
}
