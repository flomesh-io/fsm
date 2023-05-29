// Package pipy implements utility routines related to Pipy proxy, and models an instance of a proxy
// to be able to generate XDS configurations for it.
package pipy

import (
	"net"
)

// NetAddr represents a network end point address.
//
// The two methods Network and String conventionally return strings
// that can be passed as the arguments to Dial, but the exact form
// and meaning of the strings is up to the implementation.
type NetAddr struct {
	address string
}

// Network implements net.Addr interface
func (a *NetAddr) Network() string {
	return "tcp"
}

// String form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
func (a *NetAddr) String() string {
	return a.address
}

// NewNetAddress creates a new net.Addr
func NewNetAddress(address string) net.Addr {
	return &NetAddr{
		address: address,
	}
}
