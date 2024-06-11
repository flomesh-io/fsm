package cache

import (
	"context"

	"github.com/jinzhu/copier"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
	"github.com/flomesh-io/fsm/pkg/gateway/status/route"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayProcessorV2) processTCPRoutes() []interface{} {
	list := &gwv1alpha2.TCPRouteList{}
	if err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayTCPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TCPRoutes: %v", err)
		return nil
	}

	routes := make([]interface{}, 0)
	for _, tcpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := route.NewRouteStatusHolder(
			tcpRoute,
			&tcpRoute.ObjectMeta,
			&tcpRoute.TypeMeta,
			nil,
			gwutils.ToSlicePtr(tcpRoute.Status.Parents),
		)

		if c.ignoreTCPRoute(tcpRoute, rsh) {
			continue
		}

		if t2 := c.toV2TCPRoute(tcpRoute); t2 != nil {
			routes = append(routes, t2)
		}
	}

	return routes
}

func (c *GatewayProcessorV2) toV2TCPRoute(tcpRoute *gwv1alpha2.TCPRoute) *v2.TCPRoute {
	t2 := &v2.TCPRoute{}
	if err := copier.CopyWithOption(t2, tcpRoute, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		log.Error().Msgf("Failed to copy TCPRoute: %v", err)
		return nil
	}

	t2.Spec.Rules = make([]v2.TCPRouteRule, 0)
	for _, rule := range tcpRoute.Spec.Rules {
		rule := rule
		if r2 := c.toV2TCPRouteRule(tcpRoute, rule); r2 != nil {
			t2.Spec.Rules = append(t2.Spec.Rules, *r2)
		}
	}

	if len(t2.Spec.Rules) == 0 {
		return nil
	}

	return t2
}

func (c *GatewayProcessorV2) toV2TCPRouteRule(tcpRoute *gwv1alpha2.TCPRoute, rule gwv1alpha2.TCPRouteRule) *v2.TCPRouteRule {
	r2 := &v2.TCPRouteRule{}
	if err := copier.CopyWithOption(r2, &rule, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		log.Error().Msgf("Failed to copy TCPRouteRule: %v", err)
		return nil
	}

	r2.BackendRefs = c.toV2TCPBackendRefs(tcpRoute, rule.BackendRefs)
	if len(r2.BackendRefs) == 0 {
		return nil
	}

	return r2
}

func (c *GatewayProcessorV2) toV2TCPBackendRefs(tcpRoute *gwv1alpha2.TCPRoute, refs []gwv1alpha2.BackendRef) []v2.BackendRef {
	backendRefs := make([]v2.BackendRef, 0)
	for _, backend := range refs {
		backend := backend
		if svcPort := c.backendRefToServicePortName(tcpRoute, backend.BackendObjectReference); svcPort != nil {
			backendRefs = append(backendRefs, v2.BackendRef{
				Kind:   "Backend",
				Name:   svcPort.String(),
				Weight: backendWeight(backend),
			})

			c.services[svcPort.String()] = serviceContextV2{
				svcPortName: *svcPort,
			}
		}
	}

	return backendRefs
}

func (c *GatewayProcessorV2) ignoreTCPRoute(tcpRoute *gwv1alpha2.TCPRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range tcpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		allowedListeners := gwutils.GetAllowedListeners(c.cache.client, c.gateway, h)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1.TLSProtocolType:
				if listener.TLS == nil {
					continue
				}

				if listener.TLS.Mode == nil {
					continue
				}

				if *listener.TLS.Mode != gwv1.TLSModeTerminate {
					continue
				}

				hostnames := gwutils.GetValidHostnames(listener.Hostname, nil)

				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}

				return false
			case gwv1.TCPProtocolType:
				return false
			}
		}
	}

	return true
}
