package cache

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"

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
	gateways := make([]*gwv1.Gateway, 0)

	allGateways, err := c.informers.GetListers().Gateway.Gateways(corev1.NamespaceAll).List(selectAll)
	if err != nil {
		return nil
	}

	for _, gw := range allGateways {
		if gwutils.IsActiveGateway(gw) {
			gw.GetObjectKind().SetGroupVersionKind(constants.GatewayGVK)
			gateways = append(gateways, gw)
		}
	}

	return gateways
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
		r := r.(*gwv1alpha2.GRPCRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if c.isRefToService(r, backend.BackendObjectReference, service) {
					return true
				}

				for _, filter := range backend.Filters {
					if filter.Type == gwv1alpha2.GRPCRouteFilterRequestMirror {
						if c.isRefToService(r, filter.RequestMirror.BackendRef, service) {
							return true
						}
					}
				}
			}

			for _, filter := range rule.Filters {
				if filter.Type == gwv1alpha2.GRPCRouteFilterRequestMirror {
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
func (c *GatewayCache) isEffectiveTargetRef(policy client.Object, targetRef gwv1alpha2.PolicyTargetReference) bool {
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
func (c *GatewayCache) isRoutableTargetService(owner client.Object, targetRef gwv1alpha2.PolicyTargetReference) bool {
	if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
		(targetRef.Group == constants.FlomeshAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
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
					if len(l.TLS.CertificateRefs) == 0 {
						continue
					}

					for _, ref := range l.TLS.CertificateRefs {
						if c.isRefToSecret(gw, ref, secret) {
							return true
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

func (c *GatewayCache) getSecretFromCache(key client.ObjectKey) (*corev1.Secret, error) {
	obj, err := c.informers.GetListers().Secret.Secrets(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(constants.SecretGVK)

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

	return service.Spec.ClusterIP == corev1.ClusterIPNone && len(service.Spec.Selector) == 0
}

func (c *GatewayCache) backendRefToServicePortName(referer client.Object, ref gwv1.BackendObjectReference) *fgw.ServicePortName {
	if !isValidBackendRefToGroupKindOfService(ref) {
		log.Error().Msgf("Unsupported backend group %s and kind %s for service", *ref.Group, *ref.Kind)
		return nil
	}

	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
		c.getResourcesFromCache(informers.ReferenceGrantResourceType, false),
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(*ref.Group),
			Kind:      string(*ref.Kind),
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
	) {
		log.Error().Msgf("Cross-namespace reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
		return nil
	}

	return &fgw.ServicePortName{
		NamespacedName: types.NamespacedName{
			Namespace: gwutils.Namespace(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		},
		Port: pointer.Int32(int32(*ref.Port)),
	}
}

func (c *GatewayCache) targetRefToServicePortName(referer client.Object, ref gwv1alpha2.PolicyTargetReference, port int32) *fgw.ServicePortName {
	if !isValidTargetRefToGroupKindOfService(ref) {
		log.Error().Msgf("Unsupported target group %s and kind %s for service", ref.Group, ref.Kind)
		return nil
	}

	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
		c.getResourcesFromCache(informers.ReferenceGrantResourceType, false),
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(ref.Group),
			Kind:      string(ref.Kind),
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
	) {
		log.Error().Msgf("Cross-namespace reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(ref.Kind), string(ref.Group), string(*ref.Namespace), ref.Name)
		return nil
	}

	return &fgw.ServicePortName{
		NamespacedName: types.NamespacedName{
			Namespace: gwutils.Namespace(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		},
		Port: pointer.Int32(port),
	}
}

func (c *GatewayCache) toFSMHTTPRouteFilter(referer client.Object, filter gwv1.HTTPRouteFilter, services map[string]serviceContext) fgw.Filter {
	result := fgw.HTTPRouteFilter{Type: filter.Type}

	if filter.RequestHeaderModifier != nil {
		result.RequestHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Add),
			Remove: filter.RequestHeaderModifier.Remove,
		}
	}

	if filter.ResponseHeaderModifier != nil {
		result.ResponseHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Add),
			Remove: filter.ResponseHeaderModifier.Remove,
		}
	}

	if filter.RequestRedirect != nil {
		result.RequestRedirect = &fgw.HTTPRequestRedirectFilter{
			Scheme:     filter.RequestRedirect.Scheme,
			Hostname:   toFSMHostname(filter.RequestRedirect.Hostname),
			Path:       toFSMHTTPPathModifier(filter.RequestRedirect.Path),
			Port:       toFSMPortNumber(filter.RequestRedirect.Port),
			StatusCode: filter.RequestRedirect.StatusCode,
		}
	}

	if filter.URLRewrite != nil {
		result.URLRewrite = &fgw.HTTPURLRewriteFilter{
			Hostname: toFSMHostname(filter.URLRewrite.Hostname),
			Path:     toFSMHTTPPathModifier(filter.URLRewrite.Path),
		}
	}

	if filter.RequestMirror != nil {
		if svcPort := c.backendRefToServicePortName(referer, filter.RequestMirror.BackendRef); svcPort != nil {
			result.RequestMirror = &fgw.HTTPRequestMirrorFilter{
				BackendService: svcPort.String(),
			}

			services[svcPort.String()] = serviceContext{
				svcPortName: *svcPort,
			}
		}
	}

	// TODO: implement it later
	if filter.ExtensionRef != nil {
		result.ExtensionRef = filter.ExtensionRef
	}

	return result
}

func (c *GatewayCache) toFSMGRPCRouteFilter(referer client.Object, filter gwv1alpha2.GRPCRouteFilter, services map[string]serviceContext) fgw.Filter {
	result := fgw.GRPCRouteFilter{Type: filter.Type}

	if filter.RequestHeaderModifier != nil {
		result.RequestHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Add),
			Remove: filter.RequestHeaderModifier.Remove,
		}
	}

	if filter.ResponseHeaderModifier != nil {
		result.ResponseHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Add),
			Remove: filter.ResponseHeaderModifier.Remove,
		}
	}

	if filter.RequestMirror != nil {
		if svcPort := c.backendRefToServicePortName(referer, filter.RequestMirror.BackendRef); svcPort != nil {
			result.RequestMirror = &fgw.HTTPRequestMirrorFilter{
				BackendService: svcPort.String(),
			}

			services[svcPort.String()] = serviceContext{
				svcPortName: *svcPort,
			}
		}
	}

	// TODO: implement it later
	if filter.ExtensionRef != nil {
		result.ExtensionRef = filter.ExtensionRef
	}

	return result
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

func (c *GatewayCache) secretRefToSecret(referer client.Object, ref gwv1.SecretObjectReference, referenceGrants []client.Object) (*corev1.Secret, error) {
	if !isValidRefToGroupKindOfSecret(ref) {
		return nil, fmt.Errorf("unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind)
	}

	// If the secret is in a different namespace than the referer, check ReferenceGrants
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
		referenceGrants,
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     corev1.GroupName,
			Kind:      constants.KubernetesSecretKind,
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
	) {
		return nil, fmt.Errorf("cross-namespace secert reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
	}

	return c.getSecretFromCache(client.ObjectKey{
		Namespace: gwutils.Namespace(ref.Namespace, referer.GetNamespace()),
		Name:      string(ref.Name),
	})
}
