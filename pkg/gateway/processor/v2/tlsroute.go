package v2

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"

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

func (c *ConfigGenerator) processTLSRoutes() []interface{} {
	list := &gwv1alpha2.TLSRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayTLSRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TLSRoutes: %v", err)
		return nil
	}

	resources := make([]interface{}, 0)
	for _, tlsRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := route.NewRouteStatusHolder(
			tlsRoute,
			&tlsRoute.ObjectMeta,
			&tlsRoute.TypeMeta,
			tlsRoute.Spec.Hostnames,
			gwutils.ToSlicePtr(tlsRoute.Status.Parents),
		)

		if c.ignoreTLSRoute(tlsRoute, rsh) {
			continue
		}

		holder := rsh.StatusUpdateFor(
			gwv1.ParentReference{
				Namespace: ptr.To(gwv1.Namespace(c.gateway.Namespace)),
				Name:      gwv1.ObjectName(c.gateway.Name),
			},
		)

		t2, bks := c.toV2TLSRoute(tlsRoute, holder)
		if t2 != nil {
			resources = append(resources, t2)
			resources = append(resources, bks...)
		}
	}

	return resources
}

func (c *ConfigGenerator) toV2TLSRoute(tlsRoute *gwv1alpha2.TLSRoute, holder status.RouteParentStatusObject) (*v2.TLSRoute, []interface{}) {
	t2 := &v2.TLSRoute{}
	if err := gwutils.DeepCopy(t2, tlsRoute); err != nil {
		log.Error().Msgf("Failed to copy TLSRoute: %v", err)
		return nil, nil
	}

	backends := make([]interface{}, 0)
	t2.Spec.Rules = make([]v2.TLSRouteRule, 0)
	for _, rule := range tlsRoute.Spec.Rules {
		rule := rule
		r2, bks := c.toV2TLSRouteRule(tlsRoute, rule, holder)
		if r2 != nil {
			t2.Spec.Rules = append(t2.Spec.Rules, *r2)
			backends = append(backends, bks...)
		}
	}

	if len(t2.Spec.Rules) == 0 {
		return nil, nil
	}

	return t2, backends
}

func (c *ConfigGenerator) toV2TLSRouteRule(tlsRoute *gwv1alpha2.TLSRoute, rule gwv1alpha2.TLSRouteRule, holder status.RouteParentStatusObject) (*v2.TLSRouteRule, []interface{}) {
	r2 := &v2.TLSRouteRule{}
	if err := gwutils.DeepCopy(r2, &rule); err != nil {
		log.Error().Msgf("Failed to copy TCPRouteRule: %v", err)
		return nil, nil
	}

	backendRefs, bks := c.toV2TLSBackendRefs(tlsRoute, rule.BackendRefs, holder)
	if len(backendRefs) == 0 {
		return nil, nil
	}

	r2.BackendRefs = backendRefs

	return r2, bks
}

func (c *ConfigGenerator) toV2TLSBackendRefs(_ *gwv1alpha2.TLSRoute, refs []gwv1alpha2.BackendRef, _ status.RouteParentStatusObject) ([]v2.BackendRef, []interface{}) {
	backendRefs := make([]v2.BackendRef, 0)
	backends := make([]interface{}, 0)

	for _, backend := range refs {
		name := fmt.Sprintf("%s%s", backend.Name, formatTLSPort(backend.Port))

		backendRefs = append(backendRefs, v2.NewBackendRefWithWeight(name, backendWeight(backend)))

		backends = append(backends, v2.NewBackend(name, []v2.BackendTarget{
			{
				Address: string(backend.Name),
				Port:    tlsBackendPort(backend.Port),
			},
		}))
	}

	return backendRefs, backends
}

func formatTLSPort(port *gwv1alpha2.PortNumber) string {
	if port == nil {
		return ""
	}

	return fmt.Sprintf("-%d", *port)
}

func tlsBackendPort(port *gwv1alpha2.PortNumber) *int32 {
	if port == nil {
		return nil
	}

	return ptr.To(int32(*port))
}

func (c *ConfigGenerator) ignoreTLSRoute(tlsRoute *gwv1alpha2.TLSRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range tlsRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		if !gwutils.IsEffectiveRouteForParent(rsh, parentRef) {
			continue
		}

		allowedListeners := gwutils.GetAllowedListeners(c.client, c.gateway, h)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			if listener.Protocol != gwv1.TLSProtocolType {
				continue
			}

			if listener.TLS == nil {
				continue
			}

			if listener.TLS.Mode == nil {
				continue
			}

			if *listener.TLS.Mode != gwv1.TLSModePassthrough {
				continue
			}

			hostnames := gwutils.GetValidHostnames(listener.Hostname, tlsRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			return false
		}
	}

	return true
}
