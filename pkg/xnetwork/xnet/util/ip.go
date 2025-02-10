package util

import (
	"encoding/binary"
	"errors"
	"net"
)

var ErrInvalidIPAddress = errors.New("invalid ip address")
var ErrNotIPv4Address = errors.New("not an IPv4 address")

func IPToInt(ipaddr net.IP) (addr0, addr1, addr2, addr4 uint32, v6 uint8, err error) {
	if ipaddr.To4() != nil {
		addr0 = binary.LittleEndian.Uint32(ipaddr.To4())
		return
	}
	if v6Bytes := ipaddr.To16(); v6Bytes != nil {
		addr0 = binary.LittleEndian.Uint32(v6Bytes[0:4])
		addr1 = binary.LittleEndian.Uint32(v6Bytes[4:8])
		addr2 = binary.LittleEndian.Uint32(v6Bytes[8:12])
		addr4 = binary.LittleEndian.Uint32(v6Bytes[12:16])
		v6 = 1
		return
	}
	err = ErrInvalidIPAddress
	return
}

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
