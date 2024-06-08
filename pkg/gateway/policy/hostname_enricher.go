package policy

import (
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/faultinjection"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

type HostnamePolicyEnricher interface {
	Enrich(hostname string, r fgw.L7RouteRuleSpec)
}

// ---

func NewRateLimitHostnameEnricher(cache cache.Cache, selector fields.Selector) HostnamePolicyEnricher {
	return &RateLimitHostnameEnricher{
		data: gwutils.SortResources(gwutils.GetRateLimitsMatchTypeHostname(cache, selector)),
	}
}

// RateLimitHostnameEnricher is an enricher for rate limit policies at the hostname level
type RateLimitHostnameEnricher struct {
	data []client.Object
}

func (e *RateLimitHostnameEnricher) Enrich(hostname string, r fgw.L7RouteRuleSpec) {
	if len(e.data) == 0 {
		return
	}

	for _, rateLimit := range e.data {
		rateLimit := rateLimit.(*gwpav1alpha1.RateLimitPolicy)
		if rl := ratelimit.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, rateLimit); rl != nil && r.GetRateLimit() == nil {
			r.SetRateLimit(newRateLimitConfig(rl))
			break
		}
	}
}

// ---

func NewAccessControlHostnameEnricher(cache cache.Cache, selector fields.Selector) HostnamePolicyEnricher {
	return &accessControlHostnameEnricher{
		data: gwutils.SortResources(gwutils.GetAccessControlsMatchTypeHostname(cache, selector)),
	}
}

// accessControlHostnameEnricher is an enricher for access control policies at the hostname level
type accessControlHostnameEnricher struct {
	data []client.Object
}

func (e *accessControlHostnameEnricher) Enrich(hostname string, r fgw.L7RouteRuleSpec) {
	if len(e.data) == 0 {
		return
	}

	for _, ac := range e.data {
		ac := ac.(*gwpav1alpha1.AccessControlPolicy)
		if cfg := accesscontrol.GetAccessControlConfigIfRouteHostnameMatchesPolicy(hostname, ac); cfg != nil && r.GetAccessControlLists() == nil {
			r.SetAccessControlLists(newAccessControlLists(cfg))
			break
		}
	}
}

// ---

func NewFaultInjectionHostnameEnricher(cache cache.Cache, selector fields.Selector) HostnamePolicyEnricher {
	return &faultInjectionHostnameEnricher{
		data: gwutils.SortResources(gwutils.GetFaultInjectionsMatchTypeHostname(cache, selector)),
	}
}

// faultInjectionHostnameEnricher is an enricher for fault injection policies at the hostname level
type faultInjectionHostnameEnricher struct {
	data []client.Object
}

func (e *faultInjectionHostnameEnricher) Enrich(hostname string, r fgw.L7RouteRuleSpec) {
	if len(e.data) == 0 {
		return
	}

	for _, fj := range e.data {
		fj := fj.(*gwpav1alpha1.FaultInjectionPolicy)
		if cfg := faultinjection.GetFaultInjectionConfigIfRouteHostnameMatchesPolicy(hostname, fj); cfg != nil && r.GetFaultInjection() == nil {
			r.SetFaultInjection(newFaultInjection(cfg))
			break
		}
	}
}
