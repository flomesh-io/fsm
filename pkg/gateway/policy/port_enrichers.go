package policy

import (
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
)

type PortPolicyEnricher interface {
	Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener)
}

// ---

func NewRateLimitPortEnricher(cache cache.Cache, selector fields.Selector) PortPolicyEnricher {
	return &rateLimitPortEnricher{
		data: gwutils.SortResources(gwutils.GetRateLimitsMatchTypePort(cache, selector)),
	}
}

// rateLimitPortEnricher is an enricher for rate limit policies at the port level
type rateLimitPortEnricher struct {
	data []client.Object
}

func (e *rateLimitPortEnricher) Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener) {
	switch listenerCfg.Protocol {
	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType, gwv1.TCPProtocolType:
		if len(e.data) == 0 {
			return
		}

		for _, rateLimit := range e.data {
			rateLimit := rateLimit.(*gwpav1alpha1.RateLimitPolicy)
			//rateLimit := rateLimit
			//if !gwutils.HasAccessToTarget(e.ReferenceGrants, &rateLimit, rateLimit.Spec.TargetRef, gw) {
			//	continue
			//}

			if len(rateLimit.Spec.Ports) == 0 {
				continue
			}

			if r := ratelimit.GetRateLimitIfPortMatchesPolicy(port, rateLimit); r != nil && listenerCfg.BpsLimit == nil {
				listenerCfg.BpsLimit = r
				break
			}
		}
	default:
		log.Warn().Msgf("rateLimitPortEnricher: unsupported protocol %s", listenerCfg.Protocol)
	}
}

// ---

func NewAccessControlPortEnricher(cache cache.Cache, selector fields.Selector) PortPolicyEnricher {
	return &accessControlPortEnricher{
		data: gwutils.SortResources(gwutils.GetAccessControlsMatchTypePort(cache, selector)),
	}
}

// accessControlPortEnricher is an enricher for access control policies at the port level
type accessControlPortEnricher struct {
	data []client.Object
}

func (e *accessControlPortEnricher) Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener) {
	switch listenerCfg.Protocol {
	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType, gwv1.TCPProtocolType, gwv1.UDPProtocolType:
		if len(e.data) == 0 {
			return
		}

		for _, accessControl := range e.data {
			accessControl := accessControl.(*gwpav1alpha1.AccessControlPolicy)
			//ac := accessControl
			//if !gwutils.HasAccessToTarget(e.ReferenceGrants, &ac, ac.Spec.TargetRef, gw) {
			//	continue
			//}

			if len(accessControl.Spec.Ports) == 0 {
				continue
			}

			if c := accesscontrol.GetAccessControlConfigIfPortMatchesPolicy(port, accessControl); c != nil && listenerCfg.AccessControlLists == nil {
				listenerCfg.AccessControlLists = newAccessControlLists(c)
				break
			}
		}
	default:
		log.Warn().Msgf("accessControlPortEnricher: unsupported protocol %s", listenerCfg.Protocol)
	}
}
