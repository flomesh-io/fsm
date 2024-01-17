package policy

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/gatewaytls"

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
	Data []gwpav1alpha1.RateLimitPolicy
}

func (e *RateLimitPortEnricher) Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener) {
	switch listenerCfg.Protocol {
	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType, gwv1.TCPProtocolType:
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
	default:
		log.Warn().Msgf("RateLimitPortEnricher: unsupported protocol %s", listenerCfg.Protocol)
	}
}

// ---

// AccessControlPortEnricher is an enricher for access control policies at the port level
type AccessControlPortEnricher struct {
	Data []gwpav1alpha1.AccessControlPolicy
}

func (e *AccessControlPortEnricher) Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener) {
	switch listenerCfg.Protocol {
	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType, gwv1.TCPProtocolType, gwv1.UDPProtocolType:
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
	default:
		log.Warn().Msgf("AccessControlPortEnricher: unsupported protocol %s", listenerCfg.Protocol)
	}
}

// ---

// GatewayTLSPortEnricher is an enricher for access control policies at the port level
type GatewayTLSPortEnricher struct {
	Data []gwpav1alpha1.GatewayTLSPolicy
}

func (e *GatewayTLSPortEnricher) Enrich(gw *gwv1.Gateway, port gwv1.PortNumber, listenerCfg *fgw.Listener) {
	switch listenerCfg.Protocol {
	case gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
		if len(e.Data) == 0 {
			return
		}

		for _, policy := range e.Data {
			if !gwutils.IsRefToTarget(policy.Spec.TargetRef, gw) {
				continue
			}

			if len(policy.Spec.Ports) == 0 {
				continue
			}

			if c := gatewaytls.GetGatewayTLSConfigIfPortMatchesPolicy(port, policy); c != nil &&
				listenerCfg.TLS != nil &&
				listenerCfg.TLS.TLSModeType == gwv1.TLSModeTerminate &&
				listenerCfg.TLS.MTLS == nil {
				// only set if TLS Mode is set to terminate
				listenerCfg.TLS.MTLS = c.MTLS
				break
			}
		}
	default:
		log.Warn().Msgf("GatewayTLSPortEnricher: unsupported protocol %s", listenerCfg.Protocol)
	}
}
