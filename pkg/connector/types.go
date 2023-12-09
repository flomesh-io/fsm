package connector

import (
	"fmt"
	"strings"
)

var (
	// ServiceSourceValue is the value of the source.
	ServiceSourceValue = "sync-from-k8s"

	// ViaGateway defines gateway settings
	ViaGateway = &Gateway{}
)

type ProtocolPort struct {
	HTTPPort uint
	GRPCPort uint
}

type Gateway struct {
	IngressIPSelector string
	EgressIPSelector  string

	IngressAddr string
	EgressAddr  string

	Ingress ProtocolPort
	Egress  ProtocolPort
}

func (gw *Gateway) Enable() bool {
	return gw.Ingress.HTTPPort > 0 || gw.Ingress.GRPCPort > 0 || gw.Egress.HTTPPort > 0 || gw.Egress.GRPCPort > 0
}

// ServiceInstanceID generates a unique ID for a service. This ID is not meant
// to be particularly human-friendly.
func ServiceInstanceID(name, addr string, port int) string {
	return strings.ToLower(fmt.Sprintf("%s-%s-%d-%s", name, addr, port, ServiceSourceValue))
}
