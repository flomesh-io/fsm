package policy

import (
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/faultinjection"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
)

type HTTPRoutePolicyEnricher interface {
	Enrich(match gwv1.HTTPRouteMatch, matchCfg *fgw.HTTPTrafficMatch)
}

// ---

func NewRateLimitHTTPRouteEnricher(cache cache.Cache, selector fields.Selector) HTTPRoutePolicyEnricher {
	return &rateLimitHTTPRouteEnricher{
		data: gwutils.SortResources(gwutils.GetRateLimitsMatchTypeHTTPRoute(cache, selector)),
	}
}

// rateLimitHTTPRouteEnricher is an enricher for rate limit policies at the HTTP route level
type rateLimitHTTPRouteEnricher struct {
	data []client.Object
}

func (e *rateLimitHTTPRouteEnricher) Enrich(match gwv1.HTTPRouteMatch, matchCfg *fgw.HTTPTrafficMatch) {
	if len(e.data) == 0 {
		return
	}

	for _, rateLimit := range e.data {
		rateLimit := rateLimit.(*gwpav1alpha1.RateLimitPolicy)

		if len(rateLimit.Spec.HTTPRateLimits) == 0 {
			continue
		}

		if r := ratelimit.GetRateLimitIfHTTPRouteMatchesPolicy(match, rateLimit); r != nil && matchCfg.RateLimit == nil {
			matchCfg.RateLimit = newRateLimitConfig(r)
			break
		}
	}
}

// ---

func NewAccessControlHTTPRouteEnricher(cache cache.Cache, selector fields.Selector) HTTPRoutePolicyEnricher {
	return &accessControlHTTPRouteEnricher{
		data: gwutils.SortResources(gwutils.GetAccessControlsMatchTypeHTTPRoute(cache, selector)),
	}
}

// accessControlHTTPRouteEnricher is an enricher for access control policies at the HTTP route level
type accessControlHTTPRouteEnricher struct {
	data []client.Object
}

func (e *accessControlHTTPRouteEnricher) Enrich(match gwv1.HTTPRouteMatch, matchCfg *fgw.HTTPTrafficMatch) {
	if len(e.data) == 0 {
		return
	}

	for _, accessControl := range e.data {
		accessControl := accessControl.(*gwpav1alpha1.AccessControlPolicy)

		if len(accessControl.Spec.HTTPAccessControls) == 0 {
			continue
		}

		if c := accesscontrol.GetAccessControlConfigIfHTTPRouteMatchesPolicy(match, accessControl); c != nil && matchCfg.AccessControlLists == nil {
			matchCfg.AccessControlLists = newAccessControlLists(c)
			break
		}
	}
}

// ---

func NewFaultInjectionHTTPRouteEnricher(cache cache.Cache, selector fields.Selector) HTTPRoutePolicyEnricher {
	return &faultInjectionHTTPRouteEnricher{
		data: gwutils.SortResources(gwutils.GetFaultInjectionsMatchTypeHTTPRoute(cache, selector)),
	}
}

// faultInjectionHTTPRouteEnricher is an enricher for fault injection policies at the HTTP route level
type faultInjectionHTTPRouteEnricher struct {
	data []client.Object
}

func (e *faultInjectionHTTPRouteEnricher) Enrich(match gwv1.HTTPRouteMatch, matchCfg *fgw.HTTPTrafficMatch) {
	if len(e.data) == 0 {
		return
	}

	for _, faultInjection := range e.data {
		faultInjection := faultInjection.(*gwpav1alpha1.FaultInjectionPolicy)

		if len(faultInjection.Spec.HTTPFaultInjections) == 0 {
			continue
		}

		if f := faultinjection.GetFaultInjectionConfigIfHTTPRouteMatchesPolicy(match, faultInjection); f != nil && matchCfg.FaultInjection == nil {
			matchCfg.FaultInjection = newFaultInjection(f)
			break
		}
	}
}
