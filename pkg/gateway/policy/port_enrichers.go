package policy

import (
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

type PortPolicyEnricher interface {
	Enrich(gw *gwv1beta1.Gateway, port gwv1beta1.PortNumber, listenerCfg *routecfg.Listener)
}

// ---

// RateLimitPortEnricher is an enricher for rate limit policies at the port level
type RateLimitPortEnricher struct {
	Data []gwpav1alpha1.RateLimitPolicy
}

func (e *RateLimitPortEnricher) Enrich(gw *gwv1beta1.Gateway, port gwv1beta1.PortNumber, listenerCfg *routecfg.Listener) {
	if len(e.Data) == 0 {
		return
	}

	for _, rateLimit := range e.Data {
		if !gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, gw) {
			continue
		}

		if len(rateLimit.Spec.Ports) == 0 {
			continue
		}

		if r := ratelimit.GetRateLimitIfPortMatchesPolicy(port, rateLimit); r != nil && listenerCfg.BpsLimit == nil {
			listenerCfg.BpsLimit = r
			break
		}
	}
}

// ---

// AccessControlPortEnricher is an enricher for access control policies at the port level
type AccessControlPortEnricher struct {
	Data []gwpav1alpha1.AccessControlPolicy
}

func (e *AccessControlPortEnricher) Enrich(gw *gwv1beta1.Gateway, port gwv1beta1.PortNumber, listenerCfg *routecfg.Listener) {
	if len(e.Data) == 0 {
		return
	}

	for _, accessControl := range e.Data {
		if !gwutils.IsRefToTarget(accessControl.Spec.TargetRef, gw) {
			continue
		}

		if len(accessControl.Spec.Ports) == 0 {
			continue
		}

		if c := accesscontrol.GetAccessControlConfigIfPortMatchesPolicy(port, accessControl); c != nil && listenerCfg.AccessControlLists == nil {
			listenerCfg.AccessControlLists = newAccessControlLists(c)
			break
		}
	}
}
