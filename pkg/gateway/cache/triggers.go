package cache

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

/**
 * This file contains the trigger functions for the GatewayCache.
 * These functions are used to roughly checking if the resources is referred to by another resource,
 * no need to check ReferenceGrants here
 * will compute with ReferenceGrants when generating configuration
 */

// no need to check ReferenceGrant here
func (c *GatewayCache) isRoutableService(service client.ObjectKey) bool {
	for _, checkRoutableFunc := range []func(client.ObjectKey) bool{
		c.isRoutableHTTPService,
		c.isRoutableGRPCService,
		c.isRoutableTLSService,
		c.isRoutableTCPService,
		c.isRoutableUDPService,
	} {
		if checkRoutableFunc(service) {
			return true
		}
	}

	return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isRoutableHTTPService(service client.ObjectKey) bool {
	list := &gwv1.HTTPRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.BackendHTTPRouteIndex, service.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
		return false
	}

	return len(list.Items) > 0

	//for _, r := range c.getResourcesFromCache(informers.HTTPRoutesResourceType, false) {
	//	r := r.(*gwv1.HTTPRoute)
	//	for _, rule := range r.Spec.Rules {
	//		for _, backend := range rule.BackendRefs {
	//			if c.isRefToService(r, backend.BackendObjectReference, service) {
	//				return true
	//			}
	//
	//			for _, filter := range backend.Filters {
	//				if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
	//					if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
	//						return true
	//					}
	//				}
	//			}
	//		}
	//
	//		for _, filter := range rule.Filters {
	//			if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
	//				if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
	//					return true
	//				}
	//			}
	//		}
	//	}
	//}

	//return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isRoutableGRPCService(service client.ObjectKey) bool {
	list := &gwv1.GRPCRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.BackendGRPCRouteIndex, service.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list GRPCRoutes: %v", err)
		return false
	}

	return len(list.Items) > 0
	//for _, r := range c.getResourcesFromCache(informers.GRPCRoutesResourceType, false) {
	//	r := r.(*gwv1.GRPCRoute)
	//	for _, rule := range r.Spec.Rules {
	//		for _, backend := range rule.BackendRefs {
	//			if c.isRefToService(r, backend.BackendObjectReference, service) {
	//				return true
	//			}
	//
	//			for _, filter := range backend.Filters {
	//				if filter.Type == gwv1.GRPCRouteFilterRequestMirror {
	//					if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
	//						return true
	//					}
	//				}
	//			}
	//		}
	//
	//		for _, filter := range rule.Filters {
	//			if filter.Type == gwv1.GRPCRouteFilterRequestMirror {
	//				if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
	//					return true
	//				}
	//			}
	//		}
	//	}
	//}
	//
	//return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isRoutableTLSService(service client.ObjectKey) bool {
	list := &gwv1alpha2.TLSRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.BackendTLSRouteIndex, service.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TLSRoutes: %v", err)
		return false
	}

	return len(list.Items) > 0
	//for _, r := range c.getResourcesFromCache(informers.TLSRoutesResourceType, false) {
	//	r := r.(*gwv1alpha2.TLSRoute)
	//	for _, rule := range r.Spec.Rules {
	//		for _, backend := range rule.BackendRefs {
	//			if c.isRefToService(r, backend.BackendObjectReference, service) {
	//				return true
	//			}
	//		}
	//	}
	//}
	//
	//return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isRoutableTCPService(service client.ObjectKey) bool {
	list := &gwv1alpha2.TCPRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.BackendTCPRouteIndex, service.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TCPRoutes: %v", err)
		return false
	}

	return len(list.Items) > 0
	//for _, r := range c.getResourcesFromCache(informers.TCPRoutesResourceType, false) {
	//	r := r.(*gwv1alpha2.TCPRoute)
	//	for _, rule := range r.Spec.Rules {
	//		for _, backend := range rule.BackendRefs {
	//			if c.isRefToService(r, backend.BackendObjectReference, service) {
	//				return true
	//			}
	//		}
	//	}
	//}
	//
	//return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isRoutableUDPService(service client.ObjectKey) bool {
	list := &gwv1alpha2.UDPRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.BackendUDPRouteIndex, service.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list UDPRoutes: %v", err)
		return false
	}

	return len(list.Items) > 0
	//for _, r := range c.getResourcesFromCache(informers.UDPRoutesResourceType, false) {
	//	r := r.(*gwv1alpha2.UDPRoute)
	//	for _, rule := range r.Spec.Rules {
	//		for _, backend := range rule.BackendRefs {
	//			if c.isRefToService(r, backend.BackendObjectReference, service) {
	//				return true
	//			}
	//		}
	//	}
	//}
	//
	//return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isEffectiveRoute(parentRefs []gwv1.ParentReference) bool {
	gateways := c.getActiveGateways()

	if len(gateways) == 0 {
		return false
	}

	for _, parentRef := range parentRefs {
		for _, gw := range gateways {
			if gwutils.IsRefToGateway(parentRef, gw) {
				return true
			}
		}
	}

	return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isEffectiveTargetRef(policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool {
	if targetRef.Group != constants.GatewayAPIGroup {
		return false
	}

	//referenceGrants := c.getReferenceGrantsFromCache()
	key := types.NamespacedName{
		Namespace: gwutils.Namespace(targetRef.Namespace, policy.GetNamespace()),
		Name:      string(targetRef.Name),
	}

	switch targetRef.Kind {
	case constants.GatewayAPIGatewayKind:
		gw := &gwv1.Gateway{}
		if err := c.client.Get(context.Background(), key, gw); err != nil {
			log.Error().Msgf("Failed to get Gateway: %v", err)
			return false
		}

		return gwutils.IsActiveGateway(gw)
		//gateways := c.getActiveGateways()
		//if len(gateways) == 0 {
		//	return false
		//}
		//
		//for _, gateway := range gateways {
		//	if gwutils.IsTargetRefToTarget(policy, targetRef, gateway) {
		//		return true
		//	}
		//}
	case constants.GatewayAPIHTTPRouteKind:
		//httproutes := c.getResourcesFromCache(informers.HTTPRoutesResourceType, false)
		route := &gwv1.HTTPRoute{}
		if err := c.client.Get(context.Background(), key, route); err != nil {
			log.Error().Msgf("Failed to get HTTPRoute: %v", err)
			return false
		}

		return true
		//list := &gwv1.HTTPRouteList{}
		//if err := c.client.List(context.Background(), list, &client.ListOptions{}); err != nil {
		//	log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
		//	return false
		//}
		//
		//httproutes := gwutils.ToSlicePtr(list.Items)
		//
		//if len(httproutes) == 0 {
		//	return false
		//}
		//
		//for _, route := range httproutes {
		//	if gwutils.IsTargetRefToTarget(policy, targetRef, route) {
		//		return true
		//	}
		//}
	case constants.GatewayAPIGRPCRouteKind:

		route := &gwv1.GRPCRoute{}
		if err := c.client.Get(context.Background(), key, route); err != nil {
			log.Error().Msgf("Failed to get GRPCRoute: %v", err)
			return false
		}

		return true
		//grpcroutes := c.getResourcesFromCache(informers.GRPCRoutesResourceType, false)
		//list := &gwv1.GRPCRouteList{}
		//if err := c.client.List(context.Background(), list, &client.ListOptions{}); err != nil {
		//	log.Error().Msgf("Failed to list GRPCRoutes: %v", err)
		//	return false
		//}
		//
		//grpcroutes := gwutils.ToSlicePtr(list.Items)
		//
		//if len(grpcroutes) == 0 {
		//	return false
		//}
		//
		//for _, route := range grpcroutes {
		//	if gwutils.IsTargetRefToTarget(policy, targetRef, route) {
		//		return true
		//	}
		//}
	}

	return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isRoutableTargetService(owner client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool {
	if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
		(targetRef.Group == constants.FlomeshMCSAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
		return c.isRoutableService(client.ObjectKey{
			Namespace: gwutils.Namespace(targetRef.Namespace, owner.GetNamespace()),
			Name:      string(targetRef.Name),
		})
	}

	return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isSecretReferred(secret client.ObjectKey) bool {
	//ctx := context.TODO()

	list := &gwv1.GatewayList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.SecretGatewayIndex, secret.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list Gateways: %v", err)
		return false
	}

	if len(list.Items) > 0 {
		return true
	}

	//for _, gw := range c.getActiveGateways() {
	//	for _, l := range gw.Spec.Listeners {
	//		switch l.Protocol {
	//		case gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
	//			if l.TLS == nil {
	//				continue
	//			}
	//
	//			if l.TLS.Mode == nil || *l.TLS.Mode == gwv1.TLSModeTerminate {
	//				if len(l.TLS.CertificateRefs) > 0 {
	//					for _, ref := range l.TLS.CertificateRefs {
	//						if c.isRefToSecret(gw, ref, secret) {
	//							return true
	//						}
	//					}
	//				}
	//
	//				if l.TLS.FrontendValidation != nil && len(l.TLS.FrontendValidation.CACertificateRefs) > 0 {
	//					for _, ref := range l.TLS.FrontendValidation.CACertificateRefs {
	//						ref := gwv1.SecretObjectReference{
	//							Group:     ptr.To(ref.Group),
	//							Kind:      ptr.To(ref.Kind),
	//							Name:      ref.Name,
	//							Namespace: ref.Namespace,
	//						}
	//
	//						if c.isRefToSecret(gw, ref, secret) {
	//							return true
	//						}
	//					}
	//				}
	//			}
	//		}
	//	}
	//}

	policies := &gwpav1alpha1.UpstreamTLSPolicyList{}
	if err := c.client.List(context.Background(), policies, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.SecretUpstreamTLSPolicyIndex, secret.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list UpstreamTLSPolicyList: %v", err)
		return false
	}

	return len(list.Items) > 0

	//for _, ut := range c.getResourcesFromCache(informers.UpstreamTLSPoliciesResourceType, false) {
	//	ut := ut.(*gwpav1alpha1.UpstreamTLSPolicy)
	//
	//	if ut.Spec.DefaultConfig != nil {
	//		if c.isRefToSecret(ut, ut.Spec.DefaultConfig.CertificateRef, secret) {
	//			return true
	//		}
	//	}
	//
	//	if len(ut.Spec.Ports) > 0 {
	//		for _, port := range ut.Spec.Ports {
	//			if port.Config == nil {
	//				continue
	//			}
	//
	//			if c.isRefToSecret(ut, port.Config.CertificateRef, secret) {
	//				return true
	//			}
	//		}
	//	}
	//}
	//
	//return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isConfigMapReferred(cm client.ObjectKey) bool {
	//ctx := context.TODO()
	list := &gwv1.GatewayList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ConfigMapGatewayIndex, cm.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list Gateways: %v", err)
		return false
	}

	return len(list.Items) > 0

	//for _, gw := range c.getActiveGateways() {
	//	for _, l := range gw.Spec.Listeners {
	//		switch l.Protocol {
	//		case gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
	//			if l.TLS == nil {
	//				continue
	//			}
	//
	//			if l.TLS.Mode == nil || *l.TLS.Mode == gwv1.TLSModeTerminate {
	//				if l.TLS.FrontendValidation == nil {
	//					continue
	//				}
	//
	//				if len(l.TLS.FrontendValidation.CACertificateRefs) == 0 {
	//					continue
	//				}
	//
	//				for _, ref := range l.TLS.FrontendValidation.CACertificateRefs {
	//					if c.isRefToConfigMap(gw, ref, cm) {
	//						return true
	//					}
	//				}
	//			}
	//		}
	//	}
	//}
	//
	//return false
}
