package connector

import (
	"fmt"
	"strings"

	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("connector")
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

	ClusterIP  string
	ExternalIP string
}

func (gw *Gateway) Enable() bool {
	return gw.Ingress.HTTPPort > 0 || gw.Ingress.GRPCPort > 0 || gw.Egress.HTTPPort > 0 || gw.Egress.GRPCPort > 0
}

var (
	ServiceInstanceIDFunc func(name, addr string, httpPort, grpcPort int) string
)

// ServiceInstanceID generates a unique ID for a service. This ID is not meant
// to be particularly human-friendly.
func ServiceInstanceID(name, addr string, httpPort, grpcPort int) string {
	if ServiceInstanceIDFunc != nil {
		return ServiceInstanceIDFunc(name, addr, httpPort, grpcPort)
	}
	if grpcPort > 0 {
		return strings.ToLower(fmt.Sprintf("%s-%s-%d-%d-%s", name, addr, httpPort, grpcPort, ServiceSourceValue))
	}
	return strings.ToLower(fmt.Sprintf("%s-%s-%d-%s", name, addr, httpPort, ServiceSourceValue))
}
