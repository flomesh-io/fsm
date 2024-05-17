package cache

import (
	"k8s.io/utils/ptr"

	"github.com/flomesh-io/fsm/pkg/k8s"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) getResourcesFromCache(resourceType informers.ResourceType, shouldSort bool) []client.Object {
	return c.informers.GetGatewayResourcesFromCache(resourceType, shouldSort)
}

func (c *GatewayCache) getActiveGateways() []*gwv1.Gateway {
	//gateways := make([]*gwv1.Gateway, 0)
	//
	//for _, gw := range c.getResourcesFromCache(informers.GatewaysResourceType, false) {
	//	gw := gw.(*gwv1.Gateway)
	//	if gwutils.IsActiveGateway(gw) {
	//		gateways = append(gateways, gw)
	//	}
	//}
	//
	//return gateways

	return gwutils.GetActiveGateways(c.getResourcesFromCache(informers.GatewaysResourceType, false))
}

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

func (c *GatewayCache) isRoutableHTTPService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(informers.HTTPRoutesResourceType, false) {
		r := r.(*gwv1.HTTPRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if c.isRefToService(r, backend.BackendObjectReference, service) {
					return true
				}

				for _, filter := range backend.Filters {
					if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
						if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
							return true
						}
					}
				}
			}

			for _, filter := range rule.Filters {
				if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
					if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableGRPCService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(informers.GRPCRoutesResourceType, false) {
		r := r.(*gwv1.GRPCRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if c.isRefToService(r, backend.BackendObjectReference, service) {
					return true
				}

				for _, filter := range backend.Filters {
					if filter.Type == gwv1.GRPCRouteFilterRequestMirror {
						if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
							return true
						}
					}
				}
			}

			for _, filter := range rule.Filters {
				if filter.Type == gwv1.GRPCRouteFilterRequestMirror {
					if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableTLSService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(informers.TLSRoutesResourceType, false) {
		r := r.(*gwv1alpha2.TLSRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if c.isRefToService(r, backend.BackendObjectReference, service) {
					return true
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableTCPService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(informers.TCPRoutesResourceType, false) {
		r := r.(*gwv1alpha2.TCPRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if c.isRefToService(r, backend.BackendObjectReference, service) {
					return true
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableUDPService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(informers.UDPRoutesResourceType, false) {
		r := r.(*gwv1alpha2.UDPRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if c.isRefToService(r, backend.BackendObjectReference, service) {
					return true
				}
			}
		}
	}

	return false
}

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
func (c *GatewayCache) isEffectiveTargetRef(policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool {
	if targetRef.Group != constants.GatewayAPIGroup {
		return false
	}

	referenceGrants := c.getResourcesFromCache(informers.ReferenceGrantResourceType, false)

	switch targetRef.Kind {
	case constants.GatewayAPIGatewayKind:
		gateways := c.getActiveGateways()
		if len(gateways) == 0 {
			return false
		}

		for _, gateway := range gateways {
			if gwutils.IsRefToTarget(referenceGrants, policy, targetRef, gateway) {
				return true
			}
		}
	case constants.GatewayAPIHTTPRouteKind:
		httproutes := c.getResourcesFromCache(informers.HTTPRoutesResourceType, false)
		if len(httproutes) == 0 {
			return false
		}

		for _, route := range httproutes {
			if gwutils.IsRefToTarget(referenceGrants, policy, targetRef, route) {
				return true
			}
		}
	case constants.GatewayAPIGRPCRouteKind:
		grpcroutes := c.getResourcesFromCache(informers.GRPCRoutesResourceType, false)
		if len(grpcroutes) == 0 {
			return false
		}

		for _, route := range grpcroutes {
			if gwutils.IsRefToTarget(referenceGrants, policy, targetRef, route) {
				return true
			}
		}
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
	for _, gw := range c.getActiveGateways() {
		for _, l := range gw.Spec.Listeners {
			switch l.Protocol {
			case gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
				if l.TLS == nil {
					continue
				}

				if l.TLS.Mode == nil || *l.TLS.Mode == gwv1.TLSModeTerminate {
					if len(l.TLS.CertificateRefs) > 0 {
						for _, ref := range l.TLS.CertificateRefs {
							if c.isRefToSecret(gw, ref, secret) {
								return true
							}
						}
					}

					if l.TLS.FrontendValidation != nil && len(l.TLS.FrontendValidation.CACertificateRefs) > 0 {
						for _, ref := range l.TLS.FrontendValidation.CACertificateRefs {
							ref := gwv1.SecretObjectReference{
								Group:     ptr.To(ref.Group),
								Kind:      ptr.To(ref.Kind),
								Name:      ref.Name,
								Namespace: ref.Namespace,
							}

							if c.isRefToSecret(gw, ref, secret) {
								return true
							}
						}
					}
				}
			}
		}
	}

	for _, ut := range c.getResourcesFromCache(informers.UpstreamTLSPoliciesResourceType, false) {
		ut := ut.(*gwpav1alpha1.UpstreamTLSPolicy)

		if ut.Spec.DefaultConfig != nil {
			if c.isRefToSecret(ut, ut.Spec.DefaultConfig.CertificateRef, secret) {
				return true
			}
		}

		if len(ut.Spec.Ports) > 0 {
			for _, port := range ut.Spec.Ports {
				if port.Config == nil {
					continue
				}

				if c.isRefToSecret(ut, port.Config.CertificateRef, secret) {
					return true
				}
			}
		}
	}

	return false
}

// no need to check ReferenceGrant here
func (c *GatewayCache) isConfigMapReferred(cm client.ObjectKey) bool {
	//ctx := context.TODO()
	for _, gw := range c.getActiveGateways() {
		for _, l := range gw.Spec.Listeners {
			switch l.Protocol {
			case gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
				if l.TLS == nil {
					continue
				}

				if l.TLS.Mode == nil || *l.TLS.Mode == gwv1.TLSModeTerminate {
					if l.TLS.FrontendValidation == nil {
						continue
					}

					if len(l.TLS.FrontendValidation.CACertificateRefs) == 0 {
						continue
					}

					for _, ref := range l.TLS.FrontendValidation.CACertificateRefs {
						if c.isRefToConfigMap(gw, ref, cm) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) getSecretFromCache(key client.ObjectKey) (*corev1.Secret, error) {
	obj, err := c.informers.GetListers().Secret.Secrets(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(constants.SecretGVK)

	return obj, nil
}

func (c *GatewayCache) getConfigMapFromCache(key client.ObjectKey) (*corev1.ConfigMap, error) {
	obj, err := c.informers.GetListers().ConfigMap.ConfigMaps(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(constants.ConfigMapGVK)

	return obj, nil
}

func (c *GatewayCache) getServiceFromCache(key client.ObjectKey) (*corev1.Service, error) {
	obj, err := c.informers.GetListers().Service.Services(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(constants.ServiceGVK)

	return obj, nil
}

func (c *GatewayCache) isHeadlessServiceWithoutSelector(key client.ObjectKey) bool {
	service, err := c.getServiceFromCache(key)
	if err != nil {
		log.Error().Msgf("failed to get service from cache: %v", err)
		return false
	}

	return k8s.IsHeadlessService(*service)
}

func (c *GatewayCache) isRefToService(referer client.Object, ref gwv1.BackendObjectReference, service client.ObjectKey) bool {
	if !isValidBackendRefToGroupKindOfService(ref) {
		log.Debug().Msgf("Unsupported backend group %s and kind %s for service", *ref.Group, *ref.Kind)
		return false
	}

	// fast-fail, not refer to the service with the same name
	if string(ref.Name) != service.Name {
		log.Debug().Msgf("Not refer to the service with the same name, ref.Name: %s, service.Name: %s", ref.Name, service.Name)
		return false
	}

	if ns := gwutils.Namespace(ref.Namespace, referer.GetNamespace()); ns != service.Namespace {
		log.Debug().Msgf("Not refer to the service with the same namespace, resolved namespace: %s, service.Namespace: %s", ns, service.Namespace)
		return false
	}

	if ref.Namespace != nil && string(*ref.Namespace) == service.Namespace && string(*ref.Namespace) != referer.GetNamespace() {
		gvk := referer.GetObjectKind().GroupVersionKind()
		return gwutils.ValidCrossNamespaceRef(
			c.getResourcesFromCache(informers.ReferenceGrantResourceType, false),
			gwtypes.CrossNamespaceFrom{
				Group:     gvk.Group,
				Kind:      gvk.Kind,
				Namespace: referer.GetNamespace(),
			},
			gwtypes.CrossNamespaceTo{
				Group:     string(*ref.Group),
				Kind:      string(*ref.Kind),
				Namespace: service.Namespace,
				Name:      service.Name,
			},
		)
	}

	log.Debug().Msgf("Found a match, ref: %s/%s, service: %s/%s", gwutils.Namespace(ref.Namespace, referer.GetNamespace()), ref.Name, service.Namespace, service.Name)
	return true
}

func (c *GatewayCache) isRefToSecret(referer client.Object, ref gwv1.SecretObjectReference, secret client.ObjectKey) bool {
	if !isValidRefToGroupKindOfSecret(ref) {
		return false
	}

	// fast-fail, not refer to the secret with the same name
	if string(ref.Name) != secret.Name {
		log.Debug().Msgf("Not refer to the secret with the same name, ref.Name: %s, secret.Name: %s", ref.Name, secret.Name)
		return false
	}

	if ns := gwutils.Namespace(ref.Namespace, referer.GetNamespace()); ns != secret.Namespace {
		log.Debug().Msgf("Not refer to the secret with the same namespace, resolved namespace: %s, secret.Namespace: %s", ns, secret.Namespace)
		return false
	}

	if ref.Namespace != nil && string(*ref.Namespace) == secret.Namespace && string(*ref.Namespace) != referer.GetNamespace() {
		return gwutils.ValidCrossNamespaceRef(
			c.getResourcesFromCache(informers.ReferenceGrantResourceType, false),
			gwtypes.CrossNamespaceFrom{
				Group:     referer.GetObjectKind().GroupVersionKind().Group,
				Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
				Namespace: referer.GetNamespace(),
			},
			gwtypes.CrossNamespaceTo{
				Group:     corev1.GroupName,
				Kind:      constants.KubernetesSecretKind,
				Namespace: secret.Namespace,
				Name:      secret.Name,
			},
		)
	}

	return true
}

func (c *GatewayCache) isRefToConfigMap(referer client.Object, ref gwv1.ObjectReference, cm client.ObjectKey) bool {
	if !isValidRefToGroupKindOfConfigMap(ref) {
		return false
	}

	// fast-fail, not refer to the cm with the same name
	if string(ref.Name) != cm.Name {
		log.Debug().Msgf("Not refer to the cm with the same name, ref.Name: %s, cm.Name: %s", ref.Name, cm.Name)
		return false
	}

	if ns := gwutils.Namespace(ref.Namespace, referer.GetNamespace()); ns != cm.Namespace {
		log.Debug().Msgf("Not refer to the cm with the same namespace, resolved namespace: %s, cm.Namespace: %s", ns, cm.Namespace)
		return false
	}

	if ref.Namespace != nil && string(*ref.Namespace) == cm.Namespace && string(*ref.Namespace) != referer.GetNamespace() {
		return gwutils.ValidCrossNamespaceRef(
			c.getResourcesFromCache(informers.ReferenceGrantResourceType, false),
			gwtypes.CrossNamespaceFrom{
				Group:     referer.GetObjectKind().GroupVersionKind().Group,
				Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
				Namespace: referer.GetNamespace(),
			},
			gwtypes.CrossNamespaceTo{
				Group:     corev1.GroupName,
				Kind:      constants.KubernetesConfigMapKind,
				Namespace: cm.Namespace,
				Name:      cm.Name,
			},
		)
	}

	return true
}
