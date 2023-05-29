package cidr

import (
	"bytes"
	"net"
	"sort"
)

// IncrIP ip increase
func IncrIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

// DecrIP ip decrease
func DecrIP(ip net.IP) {
	length := len(ip)
	for i := length - 1; i >= 0; i-- {
		ip[length-1]--
		if ip[length-1] < 0xFF {
			break
		}
		for j := 1; j < length; j++ {
			ip[length-j-1]--
			if ip[length-j-1] < 0xFF {
				return
			}
		}
	}
}

// Compare returns an integer comparing two ip
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func Compare(a, b net.IP) int {
	return bytes.Compare(a, b)
}

// CompareCIDR returns an integer comparing two CIDR
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func CompareCIDR(a, b *CIDR) int {
	i1, _ := a.ipnet.Mask.Size()
	j1, _ := b.ipnet.Mask.Size()
	if i1 == j1 {
		return Compare(a.ipnet.IP, b.ipnet.IP)
	}
	if i1 > j1 {
		return 1
	}
	return -1
}

// AscSortCIDRs sort cidr slice order by ip,mask asc
func AscSortCIDRs(cs []*CIDR) {
	sort.Slice(cs, func(i, j int) bool {
		i1, _ := cs[i].ipnet.Mask.Size()
		j1, _ := cs[j].ipnet.Mask.Size()
		if i1 == j1 {
			n := Compare(cs[i].ipnet.IP, cs[j].ipnet.IP)
			return n < 0
		}
		return i1 < j1
	})
}

// DescSortCIDRs sort cidr slice order by ip,mask desc
func DescSortCIDRs(cs []*CIDR) {
	sort.Slice(cs, func(i, j int) bool {
		i1, _ := cs[i].ipnet.Mask.Size()
		j1, _ := cs[j].ipnet.Mask.Size()
		if i1 == j1 {
			n := Compare(cs[i].ipnet.IP, cs[j].ipnet.IP)
			return n >= 0
		}
		return i1 > j1
	})
}
