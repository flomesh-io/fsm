package cache

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"

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

func (c *GatewayProcessorV2) processTLSRoutes() []interface{} {
	list := &gwv1alpha2.TLSRouteList{}
	if err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayTLSRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TLSRoutes: %v", err)
		return nil
	}

	routes := make([]interface{}, 0)
	backends := make([]interface{}, 0)
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

		t2 := &v2.TLSRoute{}
		err := copier.CopyWithOption(t2, tlsRoute, copier.Option{IgnoreEmpty: true, DeepCopy: true})
		if err != nil {
			log.Error().Msgf("Failed to copy TLSRoute: %v", err)
			continue
		}

		t2.Spec.Rules = make([]v2.TLSRouteRule, 0)
		for _, rule := range tlsRoute.Spec.Rules {
			rule := rule
			r2 := &v2.TLSRouteRule{}
			if err := copier.CopyWithOption(r2, &rule, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
				log.Error().Msgf("Failed to copy TCPRouteRule: %v", err)
				continue
			}

			r2.BackendRefs = make([]v2.BackendRef, 0)
			for _, backend := range rule.BackendRefs {
				name := fmt.Sprintf("%s%s", backend.Name, formatTLSPort(backend.Port))

				r2.BackendRefs = append(r2.BackendRefs, v2.BackendRef{
					Kind:   "Backend",
					Name:   name,
					Weight: backendWeight(backend),
				})
				backends = append(backends, v2.Backend{
					Kind: "Backend",
					ObjectMeta: v2.ObjectMeta{
						Name: name,
					},
					Spec: v2.BackendSpec{
						Targets: []v2.BackendTarget{
							{
								Address: string(backend.Name),
								Port:    tlsBackendPort(backend.Port),
							},
						},
					},
				})
			}

			if len(r2.BackendRefs) == 0 {
				continue
			}

			t2.Spec.Rules = append(t2.Spec.Rules, *r2)
		}

		if len(t2.Spec.Rules) == 0 {
			continue
		}

		routes = append(routes, t2)
	}

	resources := make([]interface{}, 0)
	resources = append(resources, routes...)
	resources = append(resources, backends...)

	return resources
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

func (c *GatewayProcessorV2) ignoreTLSRoute(tlsRoute *gwv1alpha2.TLSRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range tlsRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		allowedListeners := gwutils.GetAllowedListeners(c.cache.client, c.gateway, h)
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
