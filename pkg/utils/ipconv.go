package utils

import (
	"encoding/binary"
	"math/big"
	"net"
	"net/netip"
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

func IPv4Tov6(ipv41 string) string {
	var ipv6 [net.IPv6len]byte
	ipv4 := net.ParseIP(ipv41)

	copy(ipv6[:], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff})
	copy(ipv6[12:], ipv4.To4())

	return netip.AddrFrom16(ipv6).String()
}
