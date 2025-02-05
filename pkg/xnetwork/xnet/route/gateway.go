package route

import (
	"fmt"
	"net"
	"runtime"
)

// ErrNoGateway is returned if a valid gateway entry was not
// found in the route table.
type ErrNoGateway struct{}

// ErrNotImplemented is returned if your operating system
// is not supported by this package. Please raise an issue
// to request support.
type ErrNotImplemented struct{}

// ErrInvalidRouteFileFormat is returned if the format
// of /proc/net/route is unexpected on Linux systems.
// Please raise an issue.
type ErrInvalidRouteFileFormat struct {
	row string
}

func (*ErrNoGateway) Error() string {
	return "no gateway found"
}

func (*ErrNotImplemented) Error() string {
	return "not implemented for OS: " + runtime.GOOS
}

func (e *ErrInvalidRouteFileFormat) Error() string {
	return fmt.Sprintf("invalid row %q in route file: doesn't have 11 fields", e.row)
}

// DiscoverGateway is the OS independent function to get the default gateway
func DiscoverGateway() (string, net.IP, error) {
	return discoverGatewayOSSpecific()
}
