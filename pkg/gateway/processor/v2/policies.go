package v2

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func (c *ConfigGenerator) getBackendPolicyProcessors(route client.Object) []BackendPolicyProcessor {
	switch route.(type) {
	case *gwv1.HTTPRoute:
		return []BackendPolicyProcessor{
			NewBackendTLSPolicyProcessor(c),
			NewBackendLBPolicyProcessor(c),
			NewHealthCheckPolicyProcessor(c),
		}
	case *gwv1.GRPCRoute:
		return []BackendPolicyProcessor{
			NewBackendTLSPolicyProcessor(c),
			NewBackendLBPolicyProcessor(c),
		}
	case *gwv1alpha2.TCPRoute:
		return []BackendPolicyProcessor{
			NewBackendTLSPolicyProcessor(c),
		}
	default:
		return nil
	}
}

func (c *ConfigGenerator) getFilterPolicyProcessors(route client.Object) []FilterPolicyProcessor {
	switch route.(type) {
	case *gwv1alpha2.TCPRoute, *gwv1alpha2.UDPRoute:
		return []FilterPolicyProcessor{
			NewRouteRuleFilterPolicyProcessor(c),
		}
	}

	return nil
}
