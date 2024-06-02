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
 * no need to check ReferenceGrants here, over reaction to check ReferenceGrants will cause performance issue
 * will compute with ReferenceGrants when generating configuration
 */

// no need to check ReferenceGrant here
// isRoutableService checks if the service is referred by HTTPRoute/GRPCRoute/TCPRoute/UDPRoute/TLSRoute backendRefs
func (c *GatewayCache) isRoutableService(service client.ObjectKey) bool {
	for _, fn := range []func(client.ObjectKey) bool{
		c.isRoutableHTTPService,
		c.isRoutableGRPCService,
		c.isRoutableTLSService,
		c.isRoutableTCPService,
		c.isRoutableUDPService,
	} {
		if fn(service) {
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
}

// no need to check ReferenceGrant here
// isEffectiveRoute checks if the route has reference to active Gateway,
func (c *GatewayCache) isEffectiveRoute(parentRefs []gwv1.ParentReference) bool {
	gateways := c.getActiveGateways()

	if len(gateways) == 0 {
		return false
	}

	for _, parentRef := range parentRefs {
		for _, gw := range gateways {
			if gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(gw)) {
				return true
			}
		}
	}

	return false
}

// no need to check ReferenceGrant here
// isEffectiveTargetRef checks if the targetRef is effective,
// it's used to check ONLY policy attachments those are targeting Gateway or HTTPRoute/GRPCRoute resources
func (c *GatewayCache) isEffectiveTargetRef(policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool {
	if targetRef.Group != constants.GatewayAPIGroup {
		return false
	}

	//referenceGrants := c.getReferenceGrantsFromCache()
	key := types.NamespacedName{
		Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.GetNamespace()),
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
	case constants.GatewayAPIHTTPRouteKind:
		route := &gwv1.HTTPRoute{}
		if err := c.client.Get(context.Background(), key, route); err != nil {
			log.Error().Msgf("Failed to get HTTPRoute: %v", err)
			return false
		}

		return true
	case constants.GatewayAPIGRPCRouteKind:
		route := &gwv1.GRPCRoute{}
		if err := c.client.Get(context.Background(), key, route); err != nil {
			log.Error().Msgf("Failed to get GRPCRoute: %v", err)
			return false
		}

		return true
	}

	return false
}

// no need to check ReferenceGrant here
// isRoutableTargetService checks if the targetRef is a valid kind of service,
// routable means it's a service that is referred by HTTPRoute/GRPCRoute/TCPRoute/UDPRoute/TLSRoute backendRefs
func (c *GatewayCache) isRoutableTargetService(owner client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool {
	if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
		(targetRef.Group == constants.FlomeshMCSAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
		return c.isRoutableService(client.ObjectKey{
			Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, owner.GetNamespace()),
			Name:      string(targetRef.Name),
		})
	}

	return false
}

// no need to check ReferenceGrant here
// isSecretReferred checks if the secret is referred by Gateway or UpstreamTLSPolicy
func (c *GatewayCache) isSecretReferred(secret client.ObjectKey) bool {
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

	policies := &gwpav1alpha1.UpstreamTLSPolicyList{}
	if err := c.client.List(context.Background(), policies, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.SecretUpstreamTLSPolicyIndex, secret.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list UpstreamTLSPolicyList: %v", err)
		return false
	}

	return len(list.Items) > 0
}

// no need to check ReferenceGrant here
// isConfigMapReferred checks if the configMap is referred by Gateway to store the configuration of gateway or CA certificates
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
}
