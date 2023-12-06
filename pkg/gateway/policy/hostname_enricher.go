package policy

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/faultinjection"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"
)

type HostnamePolicyEnricher interface {
	Enrich(hostname string, r fgw.L7RouteRuleSpec)
}

// ---

// RateLimitHostnameEnricher is an enricher for rate limit policies at the hostname level
type RateLimitHostnameEnricher struct {
	Data []gwpav1alpha1.RateLimitPolicy
}

func (e *RateLimitHostnameEnricher) Enrich(hostname string, r fgw.L7RouteRuleSpec) {
	if len(e.Data) == 0 {
		return
	}

	for _, rateLimit := range e.Data {
		if rl := ratelimit.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, rateLimit); rl != nil && r.GetRateLimit() == nil {
			r.SetRateLimit(newRateLimitConfig(rl))
			break
		}
	}
}

// ---

// AccessControlHostnameEnricher is an enricher for access control policies at the hostname level
type AccessControlHostnameEnricher struct {
	Data []gwpav1alpha1.AccessControlPolicy
}

func (e *AccessControlHostnameEnricher) Enrich(hostname string, r fgw.L7RouteRuleSpec) {
	if len(e.Data) == 0 {
		return
	}

	for _, ac := range e.Data {
		if cfg := accesscontrol.GetAccessControlConfigIfRouteHostnameMatchesPolicy(hostname, ac); cfg != nil && r.GetAccessControlLists() == nil {
			r.SetAccessControlLists(newAccessControlLists(cfg))
			break
		}
	}
}

// ---

// FaultInjectionHostnameEnricher is an enricher for fault injection policies at the hostname level
type FaultInjectionHostnameEnricher struct {
	Data []gwpav1alpha1.FaultInjectionPolicy
}

func (e *FaultInjectionHostnameEnricher) Enrich(hostname string, r fgw.L7RouteRuleSpec) {
	if len(e.Data) == 0 {
		return
	}

	for _, fj := range e.Data {
		if cfg := faultinjection.GetFaultInjectionConfigIfRouteHostnameMatchesPolicy(hostname, fj); cfg != nil && r.GetFaultInjection() == nil {
			r.SetFaultInjection(newFaultInjection(cfg))
			break
		}
	}
}
