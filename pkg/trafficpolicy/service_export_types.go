package trafficpolicy

import policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"

// ServiceExportTrafficPolicy defines the export service policy
type ServiceExportTrafficPolicy struct {
	TrafficMatches    []*ServiceExportTrafficMatch
	HTTPRoutePolicies []*InboundTrafficPolicy
}

// ServiceExportTrafficMatch defines the attributes to match exported service traffic for a given backend
type ServiceExportTrafficMatch struct {
	Name           string
	Port           uint32
	Protocol       string
	SourceIPRanges []string
	TLS            *policyv1alpha1.TLSSpec
}
