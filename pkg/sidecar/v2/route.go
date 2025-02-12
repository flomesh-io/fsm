package v2

import (
	"net"

	"github.com/libp2p/go-netroute"

	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/route"
)

func (s *Server) matchRoute(dst string) (*net.Interface, net.HardwareAddr, error) {
	nr, err := netroute.New()
	if err != nil {
		return nil, nil, err
	}

	iface, gateway, preferredSrc, err := nr.Route(net.ParseIP(dst))
	if err != nil {
		return nil, nil, err
	}

	if gateway != nil {
		hwAddr, err := route.ARPing(preferredSrc, gateway, iface)
		if err != nil {
			return nil, nil, err
		}
		return iface, hwAddr, nil
	} else {
		hwAddr, err := route.ARPing(preferredSrc, net.ParseIP(dst), iface)
		if err != nil {
			return nil, nil, err
		}
		return iface, hwAddr, nil
	}
}
