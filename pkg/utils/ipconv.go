package utils

import (
	"net"
	"net/netip"
	"strings"
)

func IPv4Tov6(ipv41 string) string {
	var ipv6 [net.IPv6len]byte
	ipv4 := net.ParseIP(ipv41)

	copy(ipv6[:], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff})
	copy(ipv6[12:], ipv4.To4())

	str := netip.AddrFrom16(ipv6).StringExpanded()

	return strings.Replace(str, "0000:0000:0000:0000:0000:ffff", "::ffff", 1)
}
