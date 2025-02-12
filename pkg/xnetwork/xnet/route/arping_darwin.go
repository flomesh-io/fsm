package route

import "net"

// ARPing sends an arp ping over interface 'iface' to 'dstIP'
func ARPing(srcIP, dstIP net.IP, iface *net.Interface) (net.HardwareAddr, error) {
	panic("Unsupported!")
}
