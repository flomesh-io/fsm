package arp

import (
	"net"
	"net/netip"

	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
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

	for _, op := range []arp.Operation{arp.OperationRequest, arp.OperationReply} {
		if pkt, err := arp.NewPacket(op, hwAddr, ip, ethernet.Broadcast, ip); err != nil {
			return err
		} else if err = c.WriteTo(pkt, ethernet.Broadcast); err != nil {
			return err
		}
	}

	return nil
}
