package arp

import (
	"net"
	"net/netip"

	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
)

// Neighbor Cache Entry States.
const (
	NUD_NONE       = 0x00
	NUD_INCOMPLETE = 0x01
	NUD_REACHABLE  = 0x02
	NUD_STALE      = 0x04
	NUD_DELAY      = 0x08
	NUD_PROBE      = 0x10
	NUD_FAILED     = 0x20
	NUD_NOARP      = 0x40
	NUD_PERMANENT  = 0x80
)

func Announce(iface, aip string, hwAddr net.HardwareAddr) error {
	// Ensure valid network interface
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return err
	}

	// Set up ARP client with socket
	c, err := arp.Dial(ifi)
	if err != nil {
		return err
	}
	defer c.Close()

	ip, err := netip.ParseAddr(aip)
	if err != nil {
		return err
	}

	for _, op := range []arp.Operation{arp.OperationReply} {
		if pkt, err := arp.NewPacket(op, hwAddr, ip, ethernet.Broadcast, ip); err != nil {
			return err
		} else if err = c.WriteTo(pkt, ethernet.Broadcast); err != nil {
			return err
		}
	}

	return nil
}
