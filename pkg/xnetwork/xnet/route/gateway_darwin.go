package route

import (
	"net"
)

func discoverGatewayOSSpecific() (string, net.IP, error) {
	return "", nil, &ErrNotImplemented{}
}
