package policy

import (
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/faultinjection"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
)

type GRPCRoutePolicyEnricher interface {
	Enrich(match gwv1.GRPCRouteMatch, matchCfg *fgw.GRPCTrafficMatch)
}

// ---

func NewRateLimitGRPCRouteEnricher(cache cache.Cache, selector fields.Selector) GRPCRoutePolicyEnricher {
	return &rateLimitGRPCRouteEnricher{
		data: gwutils.SortResources(gwutils.GetRateLimitsMatchTypeGRPCRoute(cache, selector)),
	}
}

// rateLimitGRPCRouteEnricher is an enricher for rate limit policies at the GRPC route level
type rateLimitGRPCRouteEnricher struct {
	data []client.Object
}

func (e *rateLimitGRPCRouteEnricher) Enrich(match gwv1.GRPCRouteMatch, matchCfg *fgw.GRPCTrafficMatch) {
	if len(e.data) == 0 {
		return
	}

	for _, rateLimit := range e.data {
		rateLimit := rateLimit.(*gwpav1alpha1.RateLimitPolicy)
		if len(rateLimit.Spec.GRPCRateLimits) == 0 {
			continue
		}

		if r := ratelimit.GetRateLimitIfGRPCRouteMatchesPolicy(match, rateLimit); r != nil && matchCfg.RateLimit == nil {
			matchCfg.RateLimit = newRateLimitConfig(r)
			break
		}
	}
}

// ---

func NewAccessControlGRPCRouteEnricher(cache cache.Cache, selector fields.Selector) GRPCRoutePolicyEnricher {
	return &accessControlGRPCRouteEnricher{
		data: gwutils.SortResources(gwutils.GetAccessControlsMatchTypeGRPCRoute(cache, selector)),
	}
}

// accessControlGRPCRouteEnricher is an enricher for access control policies at the GRPC route level
type accessControlGRPCRouteEnricher struct {
	data []client.Object
}

func (e *accessControlGRPCRouteEnricher) Enrich(match gwv1.GRPCRouteMatch, matchCfg *fgw.GRPCTrafficMatch) {
	if len(e.data) == 0 {
		return
	}

	for _, accessControl := range e.data {
		accessControl := accessControl.(*gwpav1alpha1.AccessControlPolicy)
		if len(accessControl.Spec.GRPCAccessControls) == 0 {
			continue
		}

		if c := accesscontrol.GetAccessControlConfigIfGRPCRouteMatchesPolicy(match, accessControl); c != nil && matchCfg.AccessControlLists == nil {
			matchCfg.AccessControlLists = newAccessControlLists(c)
			break
		}
	}
}

// ---

func NewFaultInjectionGRPCRouteEnricher(cache cache.Cache, selector fields.Selector) GRPCRoutePolicyEnricher {
	return &faultInjectionGRPCRouteEnricher{
		data: gwutils.SortResources(gwutils.GetFaultInjectionsMatchTypeGRPCRoute(cache, selector)),
	}
}

// faultInjectionGRPCRouteEnricher is an enricher for fault injection policies at the GRPC route level
type faultInjectionGRPCRouteEnricher struct {
	data []client.Object
}

func (e *faultInjectionGRPCRouteEnricher) Enrich(match gwv1.GRPCRouteMatch, matchCfg *fgw.GRPCTrafficMatch) {
	if len(e.data) == 0 {
		return
	}

	for _, faultInjection := range e.data {
		faultInjection := faultInjection.(*gwpav1alpha1.FaultInjectionPolicy)
		if len(faultInjection.Spec.GRPCFaultInjections) == 0 {
			continue
		}

		if f := faultinjection.GetFaultInjectionConfigIfGRPCRouteMatchesPolicy(match, faultInjection); f != nil && matchCfg.FaultInjection == nil {
			matchCfg.FaultInjection = newFaultInjection(f)
			break
		}
	}
}
