package cache

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/flomesh-io/fsm/pkg/gateway/policy"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *GatewayProcessor) getPortPolicyEnrichers(gateway *gwv1.Gateway) []policy.PortPolicyEnricher {
	cc := c.cache.client
	key := client.ObjectKeyFromObject(gateway).String()
	selector := fields.OneTermEqualSelector(constants.PortPolicyAttachmentIndex, key)

	return []policy.PortPolicyEnricher{
		policy.NewRateLimitPortEnricher(cc, selector),
		policy.NewAccessControlPortEnricher(cc, selector),
	}
}

func (c *GatewayProcessor) getHostnamePolicyEnrichers(route client.Object) []policy.HostnamePolicyEnricher {
	cc := c.cache.client
	key := fmt.Sprintf("%s/%s/%s", route.GetObjectKind().GroupVersionKind().Kind, route.GetNamespace(), route.GetName())
	selector := fields.OneTermEqualSelector(constants.HostnamePolicyAttachmentIndex, key)

	return []policy.HostnamePolicyEnricher{
		policy.NewRateLimitHostnameEnricher(cc, selector),
		policy.NewAccessControlHostnameEnricher(cc, selector),
		policy.NewFaultInjectionHostnameEnricher(cc, selector),
	}
}

func (c *GatewayProcessor) getHTTPRoutePolicyEnrichers(route *gwv1.HTTPRoute) []policy.HTTPRoutePolicyEnricher {
	cc := c.cache.client
	key := client.ObjectKeyFromObject(route).String()
	selector := fields.OneTermEqualSelector(constants.HTTPRoutePolicyAttachmentIndex, key)

	return []policy.HTTPRoutePolicyEnricher{
		policy.NewRateLimitHTTPRouteEnricher(cc, selector),
		policy.NewAccessControlHTTPRouteEnricher(cc, selector),
		policy.NewFaultInjectionHTTPRouteEnricher(cc, selector),
	}
}

func (c *GatewayProcessor) getGRPCRoutePolicyEnrichers(route *gwv1.GRPCRoute) []policy.GRPCRoutePolicyEnricher {
	cc := c.cache.client
	key := client.ObjectKeyFromObject(route).String()
	selector := fields.OneTermEqualSelector(constants.GRPCRoutePolicyAttachmentIndex, key)

	return []policy.GRPCRoutePolicyEnricher{
		policy.NewRateLimitGRPCRouteEnricher(cc, selector),
		policy.NewAccessControlGRPCRouteEnricher(cc, selector),
		policy.NewFaultInjectionGRPCRouteEnricher(cc, selector),
	}
}

func (c *GatewayProcessor) getServicePolicyEnrichers(svc *corev1.Service) []policy.ServicePolicyEnricher {
	cc := c.cache.client
	key := client.ObjectKeyFromObject(svc).String()
	selector := fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, key)

	return []policy.ServicePolicyEnricher{
		policy.NewSessionStickyPolicyEnricher(cc, selector, c.targetRefToServicePortName),
		policy.NewLoadBalancerPolicyEnricher(cc, selector, c.targetRefToServicePortName),
		policy.NewCircuitBreakingPolicyEnricher(cc, selector, c.targetRefToServicePortName),
		policy.NewUpstreamTLSPolicyEnricher(cc, selector, c.targetRefToServicePortName, c.secretRefToSecret),
		policy.NewRetryPolicyEnricher(cc, selector, c.targetRefToServicePortName),
		policy.NewHealthCheckPolicyEnricher(cc, selector, c.targetRefToServicePortName),
	}
}
