package policy

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

type GRPCRoutePolicyEnricher interface {
	Enrich(match gwv1alpha2.GRPCRouteMatch, matchCfg *routecfg.GRPCTrafficMatch)
}

// ---

// RateLimitGRPCRouteEnricher is an enricher for rate limit policies at the GRPC route level
type RateLimitGRPCRouteEnricher struct {
	Data []gwpav1alpha1.RateLimitPolicy
}

func (e *RateLimitGRPCRouteEnricher) Enrich(match gwv1alpha2.GRPCRouteMatch, matchCfg *routecfg.GRPCTrafficMatch) {
	log.Debug().Msgf("RateLimitGRPCRouteEnricher.Enrich: Data=%v", e.Data)

	for _, rateLimit := range e.Data {
		if len(rateLimit.Spec.GRPCRateLimits) == 0 {
			continue
		}

		if r := gwutils.GetRateLimitIfGRPCRouteMatchesPolicy(match, rateLimit); r != nil && matchCfg.RateLimit == nil {
			matchCfg.RateLimit = newRateLimitConfig(r)
			break
		}
	}
}

// ---

// AccessControlGRPCRouteEnricher is an enricher for access control policies at the GRPC route level
type AccessControlGRPCRouteEnricher struct {
	Data []gwpav1alpha1.AccessControlPolicy
}

func (e *AccessControlGRPCRouteEnricher) Enrich(match gwv1alpha2.GRPCRouteMatch, matchCfg *routecfg.GRPCTrafficMatch) {
	log.Debug().Msgf("AccessControlGRPCRouteEnricher.Enrich: Data=%v", e.Data)

	for _, accessControl := range e.Data {
		if len(accessControl.Spec.GRPCAccessControls) == 0 {
			continue
		}

		if c := gwutils.GetAccessControlConfigIfGRPCRouteMatchesPolicy(match, accessControl); c != nil && matchCfg.AccessControlLists == nil {
			matchCfg.AccessControlLists = newAccessControlLists(c)
			break
		}
	}
}

// ---

// FaultInjectionGRPCRouteEnricher is an enricher for fault injection policies at the GRPC route level
type FaultInjectionGRPCRouteEnricher struct {
	Data []gwpav1alpha1.FaultInjectionPolicy
}

func (e *FaultInjectionGRPCRouteEnricher) Enrich(match gwv1alpha2.GRPCRouteMatch, matchCfg *routecfg.GRPCTrafficMatch) {
	log.Debug().Msgf("FaultInjectionGRPCRouteEnricher.Enrich: Data=%v", e.Data)

	for _, faultInjection := range e.Data {
		if len(faultInjection.Spec.GRPCFaultInjections) == 0 {
			continue
		}

		if f := gwutils.GetFaultInjectionConfigIfGRPCRouteMatchesPolicy(match, faultInjection); f != nil && matchCfg.FaultInjection == nil {
			matchCfg.FaultInjection = newFaultInjection(f)
			break
		}
	}
}
