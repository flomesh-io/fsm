package policy

import (
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/faultinjection"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
)

type HTTPRoutePolicyEnricher interface {
	Enrich(match gwv1beta1.HTTPRouteMatch, matchCfg *routecfg.HTTPTrafficMatch)
}

// ---

// RateLimitHTTPRouteEnricher is an enricher for rate limit policies at the HTTP route level
type RateLimitHTTPRouteEnricher struct {
	Data []gwpav1alpha1.RateLimitPolicy
}

func (e *RateLimitHTTPRouteEnricher) Enrich(match gwv1beta1.HTTPRouteMatch, matchCfg *routecfg.HTTPTrafficMatch) {
	if len(e.Data) == 0 {
		return
	}

	for _, rateLimit := range e.Data {
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

// AccessControlHTTPRouteEnricher is an enricher for access control policies at the HTTP route level
type AccessControlHTTPRouteEnricher struct {
	Data []gwpav1alpha1.AccessControlPolicy
}

func (e *AccessControlHTTPRouteEnricher) Enrich(match gwv1beta1.HTTPRouteMatch, matchCfg *routecfg.HTTPTrafficMatch) {
	if len(e.Data) == 0 {
		return
	}

	for _, accessControl := range e.Data {
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

// FaultInjectionHTTPRouteEnricher is an enricher for fault injection policies at the HTTP route level
type FaultInjectionHTTPRouteEnricher struct {
	Data []gwpav1alpha1.FaultInjectionPolicy
}

func (e *FaultInjectionHTTPRouteEnricher) Enrich(match gwv1beta1.HTTPRouteMatch, matchCfg *routecfg.HTTPTrafficMatch) {
	if len(e.Data) == 0 {
		return
	}

	for _, faultInjection := range e.Data {
		if len(faultInjection.Spec.HTTPFaultInjections) == 0 {
			continue
		}

		if f := faultinjection.GetFaultInjectionConfigIfHTTPRouteMatchesPolicy(match, faultInjection); f != nil && matchCfg.FaultInjection == nil {
			matchCfg.FaultInjection = newFaultInjection(f)
			break
		}
	}
}
