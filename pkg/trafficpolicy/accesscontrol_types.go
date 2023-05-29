package trafficpolicy

import policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"

// AccessControlTrafficPolicy defines the access control traffic match and routes for a given backend
type AccessControlTrafficPolicy struct {
	TrafficMatches    []*AccessControlTrafficMatch
	HTTPRoutePolicies []*InboundTrafficPolicy
}

// AccessControlTrafficMatch defines the attributes to match access control traffic for a given backend
type AccessControlTrafficMatch struct {
	Name           string
	Port           uint32
	Protocol       string
	SourceIPRanges []string
	TLS            *policyv1alpha1.TLSSpec

	// RateLimit defines the rate limiting policy applied for this TrafficMatch
	// +optional
	RateLimit *policyv1alpha1.RateLimitSpec
}
