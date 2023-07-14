package utils

import (
	"encoding/binary"
	"math/big"
	"net"
)

// IP2Int converts ip addr to int.
func IP2Int(ip net.IP) *big.Int {
	i := big.NewInt(0)
	i.SetBytes(ip)
	return i
}

// Int2IP4 converts uint32 to ipv4.
func Int2IP4(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

// Int2IP16 converts uint64 to ipv6.
func Int2IP16(nn uint64) net.IP {
	ip := make(net.IP, 16)
	binary.BigEndian.PutUint64(ip, nn)
	return ip
}
