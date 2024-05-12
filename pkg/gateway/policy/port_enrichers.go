package policy

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

type PortPolicyEnricher interface {
	Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener)
}

// ---

// RateLimitPortEnricher is an enricher for rate limit policies at the port level
type RateLimitPortEnricher struct {
	Data            []gwpav1alpha1.RateLimitPolicy
	ReferenceGrants []client.Object
}

func (e *RateLimitPortEnricher) Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener) {
	switch listenerCfg.Protocol {
	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType, gwv1.TCPProtocolType:
		if len(e.Data) == 0 {
			return
		}

		for _, rateLimit := range e.Data {
			rateLimit := rateLimit
			if !gwutils.IsRefToTarget(e.ReferenceGrants, &rateLimit, rateLimit.Spec.TargetRef, gw) {
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
	default:
		log.Warn().Msgf("RateLimitPortEnricher: unsupported protocol %s", listenerCfg.Protocol)
	}
}

// ---

// AccessControlPortEnricher is an enricher for access control policies at the port level
type AccessControlPortEnricher struct {
	Data            []gwpav1alpha1.AccessControlPolicy
	ReferenceGrants []client.Object
}

func (e *AccessControlPortEnricher) Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener) {
	switch listenerCfg.Protocol {
	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType, gwv1.TCPProtocolType, gwv1.UDPProtocolType:
		if len(e.Data) == 0 {
			return
		}

		for _, accessControl := range e.Data {
			ac := accessControl
			if !gwutils.IsRefToTarget(e.ReferenceGrants, &ac, ac.Spec.TargetRef, gw) {
				continue
			}

			if len(ac.Spec.Ports) == 0 {
				continue
			}

			if c := accesscontrol.GetAccessControlConfigIfPortMatchesPolicy(port, ac); c != nil && listenerCfg.AccessControlLists == nil {
				listenerCfg.AccessControlLists = newAccessControlLists(c)
				break
			}
		}
	default:
		log.Warn().Msgf("AccessControlPortEnricher: unsupported protocol %s", listenerCfg.Protocol)
	}
}
