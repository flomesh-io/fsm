package neigh

import (
	"errors"
	"fmt"
	"net"
	"syscall"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

const (
	// Option Length, 8-bit unsigned integer. The length of the option (including the type and length fields) in units of 8 octets.
	// The value 0 is invalid. Nodes MUST silently discard a ND packet that contains an option with length zero.
	// https://datatracker.ietf.org/doc/html/rfc4861
	ndpOptionLen = 1

	// ndpOptionType
	// 	Option Name                             Type
	//
	// Source Link-Layer Address                    1
	// Target Link-Layer Address                    2
	// Prefix Information                           3
	// Redirected Header                            4
	// MTU                                          5
	ndpOptionType = 2

	// Minimum byte length values for each type of valid Message.
	naLen = 20

	// Hop limit is always 255, refer RFC 4861.
	hopLimit = 255
)

// gratuitousNDPOverIface sends an unsolicited Neighbor Advertisement ICMPv6 multicast packet,
// over interface 'iface' from 'eip', announcing a given IPv6 address('eip') to all IPv6 nodes as per RFC4861.
func gratuitousNDPOverIface(eip net.IP, ifName string, macAddr net.HardwareAddr) error {
	mb, err := newNDPNeighborAdvertisementMessage(eip, macAddr)
	if err != nil {
		return fmt.Errorf("new NDP Neighbor Advertisement Message error: %v", err)
	}

	sock, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_ICMPV6)
	if err != nil {
		return err
	}
	defer syscall.Close(sock)

	if err := syscall.BindToDevice(sock, ifName); err != nil {
		return errors.New("bind-err")
	}

	syscall.SetsockoptInt(sock, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_HOPS, hopLimit)

	var r [16]byte
	copy(r[:], net.IPv6linklocalallnodes.To16())
	toSockAddrInet6 := syscall.SockaddrInet6{Addr: r}
	if err := syscall.Sendto(sock, mb, 0, &toSockAddrInet6); err != nil {
		return err
	}
	return nil
}

func newNDPNeighborAdvertisementMessage(targetAddress net.IP, hwa net.HardwareAddr) ([]byte, error) {
	naMsgBytes := make([]byte, naLen)
	naMsgBytes[0] |= 1 << 5
	copy(naMsgBytes[4:], targetAddress)

	if 1+1+len(hwa) != int(ndpOptionLen*8) {
		return nil, fmt.Errorf("hardwareAddr length error: %s", hwa)
	}
	optionsBytes := make([]byte, ndpOptionLen*8)
	optionsBytes[0] = ndpOptionType
	optionsBytes[1] = ndpOptionLen
	copy(optionsBytes[2:], hwa)
	naMsgBytes = append(naMsgBytes, optionsBytes...)

	im := icmp.Message{
		// ICMPType = 136, Neighbor Advertisement
		Type: ipv6.ICMPTypeNeighborAdvertisement,
		// Always zero.
		Code: 0,
		// The ICMP checksum. Calculated by caller or OS.
		Checksum: 0,
		Body: &icmp.RawBody{
			Data: naMsgBytes,
		},
	}
	return im.Marshal(nil)
}
