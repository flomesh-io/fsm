package v2

import (
	"context"
	"fmt"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

/**
 * This file contains the trigger functions for the GatewayProcessor.
 * These functions are used to roughly checking if the resources is referred to by another resource,
 * no need to check ReferenceGrants here, over reaction to check ReferenceGrants will cause performance issue
 * will compute with ReferenceGrants when generating configuration
 */

// no need to check ReferenceGrant here
// IsRoutableService checks if the service is referred by HTTPRoute/GRPCRoute/TCPRoute/UDPRoute/TLSRoute backendRefs

func (c *GatewayProcessor) IsRoutableService(service client.ObjectKey) bool {
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
func (c *GatewayProcessor) isRoutableHTTPService(service client.ObjectKey) bool {
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
func (c *GatewayProcessor) isRoutableGRPCService(service client.ObjectKey) bool {
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
func (c *GatewayProcessor) isRoutableTLSService(service client.ObjectKey) bool {
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
func (c *GatewayProcessor) isRoutableTCPService(service client.ObjectKey) bool {
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
func (c *GatewayProcessor) isRoutableUDPService(service client.ObjectKey) bool {
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
// IsEffectiveRoute checks if the route has reference to active Gateway,

func (c *GatewayProcessor) IsEffectiveRoute(parentRefs []gwv1.ParentReference) bool {
	gateways := gwutils.GetActiveGateways(c.client)

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
// IsEffectiveTargetRef checks if the targetRef is effective,
// it's used to check ONLY policy attachments those are targeting Gateway or HTTPRoute/GRPCRoute resources

func (c *GatewayProcessor) IsEffectiveTargetRef(policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool {
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
// IsRoutableTargetService checks if the targetRef is a valid kind of service,
// routable means it's a service that is referred by HTTPRoute/GRPCRoute/TCPRoute/UDPRoute/TLSRoute backendRefs

func (c *GatewayProcessor) IsRoutableTargetService(owner client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool {
	if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
		(targetRef.Group == constants.FlomeshMCSAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
		return c.IsRoutableService(client.ObjectKey{
			Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, owner.GetNamespace()),
			Name:      string(targetRef.Name),
		})
	}

	return false
}

// no need to check ReferenceGrant here
// IsSecretReferred checks if the secret is referred by Gateway or UpstreamTLSPolicy

func (c *GatewayProcessor) IsSecretReferred(secret client.ObjectKey) bool {
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

	policies := &gwv1alpha3.BackendTLSPolicyList{}
	if err := c.client.List(context.Background(), policies, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.SecretBackendTLSPolicyIndex, secret.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list BackendTLSPolicyList: %v", err)
		return false
	}

	return len(policies.Items) > 0
}

// no need to check ReferenceGrant here
// IsConfigMapReferred checks if the configMap is referred by Gateway to store the configuration of gateway or CA certificates

func (c *GatewayProcessor) IsConfigMapReferred(cm client.ObjectKey) bool {
	//ctx := context.TODO()
	list := &gwv1.GatewayList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ConfigMapGatewayIndex, cm.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list Gateways: %v", err)
		return false
	}

	if len(list.Items) > 0 {
		return true
	}

	policies := &gwv1alpha3.BackendTLSPolicyList{}
	if err := c.client.List(context.Background(), policies, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ConfigmapBackendTLSPolicyIndex, cm.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list BackendTLSPolicyList: %v", err)
		return false
	}

	return len(policies.Items) > 0
}

// IsFilterReferred checks if the configMap is referred by Gateway to store the configuration of gateway or CA certificates

func (c *GatewayProcessor) IsFilterReferred(filter client.ObjectKey) bool {
	for _, fn := range []func(client.ObjectKey) bool{
		c.isFilterReferredByHTTPRoute,
		c.isFilterReferredByGRPCRoute,
	} {
		if fn(filter) {
			return true
		}
	}

	return false
}

func (c *GatewayProcessor) isFilterReferredByHTTPRoute(filter client.ObjectKey) bool {
	list := &gwv1.HTTPRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ExtensionFilterHTTPRouteIndex, filter.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
		return false
	}

	return len(list.Items) > 0
}

func (c *GatewayProcessor) isFilterReferredByGRPCRoute(filter client.ObjectKey) bool {
	list := &gwv1.GRPCRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ExtensionFilterGRPCRouteIndex, filter.String()),
	}); err != nil {
		log.Error().Msgf("Failed to list GRPCRoutes: %v", err)
		return false
	}

	return len(list.Items) > 0
}

// IsListenerFilterReferred checks if the ListenerFilter is referred by Gateway

func (c *GatewayProcessor) IsListenerFilterReferred(filter client.ObjectKey) bool {
	list := &gwv1.GatewayList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ListenerFilterGatewayIndex, filter.String()),
		Namespace:     filter.Namespace,
	}); err != nil {
		return false
	}

	return len(list.Items) > 0
}

func (c *GatewayProcessor) IsFilterDefinitionReferred(def client.ObjectKey) bool {
	filters := &extv1alpha1.FilterList{}
	if err := c.client.List(context.Background(), filters, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.FilterDefinitionFilterIndex, def.String()),
	}); err != nil {
		return false
	}

	if len(filters.Items) > 0 {
		return true
	}

	list := &extv1alpha1.ListenerFilterList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.FilterDefinitionListenerFilterIndex, def.String()),
	}); err != nil {
		return false
	}

	return len(list.Items) > 0
}

func (c *GatewayProcessor) IsFilterConfigReferred(kind string, config client.ObjectKey) bool {
	filters := &extv1alpha1.FilterList{}
	if err := c.client.List(context.Background(), filters, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ConfigFilterIndex, fmt.Sprintf("%s/%s/%s", kind, config.Namespace, config.Name)),
	}); err != nil {
		return false
	}

	if len(filters.Items) > 0 {
		return true
	}

	list := &extv1alpha1.ListenerFilterList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ConfigListenerFilterIndex, fmt.Sprintf("%s/%s/%s", kind, config.Namespace, config.Name)),
	}); err != nil {
		return false
	}

	return len(list.Items) > 0
}

func (c *GatewayProcessor) IsHeadlessServiceWithoutSelector(key client.ObjectKey) bool {
	service, err := c.getServiceFromCache(key)
	if err != nil {
		log.Warn().Msgf("failed to get service from processor: %v", err)
		return false
	}

	return isHeadlessServiceWithoutSelector(service)
}

func (c *GatewayProcessor) getServiceFromCache(key client.ObjectKey) (*corev1.Service, error) {
	obj := &corev1.Service{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *GatewayProcessor) getTCPRouteFromCache(key client.ObjectKey) (*gwv1alpha2.TCPRoute, error) {
	obj := &gwv1alpha2.TCPRoute{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *GatewayProcessor) getUDPRouteFromCache(key client.ObjectKey) (*gwv1alpha2.UDPRoute, error) {
	obj := &gwv1alpha2.UDPRoute{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *GatewayProcessor) IsRoutableNamespacedTargetServices(policy client.Object, targetRefs []gwv1alpha2.NamespacedPolicyTargetReference) bool {
	for _, targetRef := range targetRefs {
		if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
			(targetRef.Group == constants.FlomeshMCSAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
			if c.IsRoutableService(types.NamespacedName{
				Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.GetNamespace()),
				Name:      string(targetRef.Name),
			}) {
				return true
			}
		}
	}

	return false
}

func (c *GatewayProcessor) IsRoutableLocalTargetServices(policy client.Object, targetRefs []gwv1alpha2.LocalPolicyTargetReference) bool {
	for _, targetRef := range targetRefs {
		if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
			(targetRef.Group == constants.FlomeshMCSAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
			if c.IsRoutableService(types.NamespacedName{
				Namespace: policy.GetNamespace(),
				Name:      string(targetRef.Name),
			}) {
				return true
			}
		}
	}

	return false
}

func (c *GatewayProcessor) IsValidLocalTargetRoutes(policy client.Object, targetRefs []gwpav1alpha2.LocalFilterPolicyTargetReference) bool {
	for _, targetRef := range targetRefs {
		switch targetRef.Kind {
		case constants.GatewayAPITCPRouteKind:
			if _, err := c.getTCPRouteFromCache(types.NamespacedName{
				Namespace: policy.GetNamespace(),
				Name:      string(targetRef.Name),
			}); err == nil {
				return true
			}
		case constants.GatewayAPIUDPRouteKind:
			if _, err := c.getUDPRouteFromCache(types.NamespacedName{
				Namespace: policy.GetNamespace(),
				Name:      string(targetRef.Name),
			}); err == nil {
				return true
			}
		}
	}

	return false
}
