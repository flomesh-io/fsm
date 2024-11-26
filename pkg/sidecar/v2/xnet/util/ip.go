package util

import (
	"encoding/binary"
	"errors"
	"net"
)

var ErrNotIPv4Address = errors.New("not an IPv4 address")

// IPv4ToInt converts IP address of version 4 from net.IP to uint32
// representation.
func IPv4ToInt(ipaddr net.IP) (uint32, error) {
	if ipaddr.To4() == nil {
		return 0, ErrNotIPv4Address
	}
	return binary.LittleEndian.Uint32(ipaddr.To4()), nil
}

// HostToNetShort converts a 16-bit integer from host to network byte order, aka "htons"
func HostToNetShort(i uint16) uint16 {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, i)
	return binary.BigEndian.Uint16(b)
}
