package trafficpolicy

import (
	"github.com/flomesh-io/fsm/pkg/endpoint"
)

// EgressGatewayPolicy is the type used to represent the egress gateway policy configurations
// applicable to a client of Egress forward.
type EgressGatewayPolicy struct {
	Global []*EgressGatewayConfig
	Rules  []*EgressGatewayRule
}

// EgressGatewayConfig is the type used to represent an egress gateway.
type EgressGatewayConfig struct {
	Service   string
	Namespace string
	Mode      string
	Weight    *int
	Endpoints []endpoint.Endpoint
}

// EgressPolicyConfig is the type used to represent an egress policy.
type EgressPolicyConfig struct {
	Name      string
	Namespace string
}

// EgressGatewayRule is the type used to represent a rule dispatching egress to gateway.
type EgressGatewayRule struct {
	Name           string
	Namespace      string
	EgressPolicies []EgressPolicyConfig
	EgressGateways []EgressGatewayConfig
}
