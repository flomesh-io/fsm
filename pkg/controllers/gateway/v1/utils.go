package v1

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func hasTCP(gateway *gwv1.Gateway) bool {
	for _, listener := range gateway.Spec.Listeners {
		switch listener.Protocol {
		case gwv1.HTTPProtocolType, gwv1.TCPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
			return true
		}
	}

	return false
}

func hasUDP(gateway *gwv1.Gateway) bool {
	for _, listener := range gateway.Spec.Listeners {
		if listener.Protocol == gwv1.UDPProtocolType {
			return true
		}
	}

	return false
}
