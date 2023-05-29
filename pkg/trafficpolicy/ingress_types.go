package trafficpolicy

import policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"

// IngressTrafficPolicy defines the ingress traffic match and routes for a given backend
type IngressTrafficPolicy struct {
	TrafficMatches    []*IngressTrafficMatch
	HTTPRoutePolicies []*InboundTrafficPolicy
}

// IngressTrafficMatch defines the attributes to match ingress traffic for a given backend
type IngressTrafficMatch struct {
	Name                     string
	Port                     uint32
	Protocol                 string
	SourceIPRanges           []string
	TLS                      *policyv1alpha1.TLSSpec
	ServerNames              []string
	SkipClientCertValidation bool

	// RateLimit defines the rate limiting policy applied for this TrafficMatch
	// +optional
	RateLimit *policyv1alpha1.RateLimitSpec
}
