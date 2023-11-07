package cache

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) isRoutableService(service client.ObjectKey) bool {
	for _, checkRoutableFunc := range []func(client.ObjectKey) bool{
		c.isRoutableHTTPService,
		c.isRoutableGRPCService,
		c.isRoutableTLSService,
		c.isRoutableTCPService,
	} {
		if checkRoutableFunc(service) {
			return true
		}
	}

	return false
}

func (c *GatewayCache) isRoutableHTTPService(service client.ObjectKey) bool {
	for key := range c.httproutes {
		// Get HTTPRoute from client-go cache
		if r, err := c.getHTTPRouteFromCache(key); err == nil {
			//r := r.(*gwv1beta1.HTTPRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}

					for _, filter := range backend.Filters {
						if filter.Type == gwv1beta1.HTTPRouteFilterRequestMirror {
							if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
								return true
							}
						}
					}
				}

				for _, filter := range rule.Filters {
					if filter.Type == gwv1beta1.HTTPRouteFilterRequestMirror {
						if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableGRPCService(service client.ObjectKey) bool {
	for key := range c.grpcroutes {
		// Get GRPCRoute from client-go cache
		if r, err := c.getGRPCRouteFromCache(key); err == nil {
			//r := r.(*gwv1alpha2.GRPCRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}

					for _, filter := range backend.Filters {
						if filter.Type == gwv1alpha2.GRPCRouteFilterRequestMirror {
							if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
								return true
							}
						}
					}
				}

				for _, filter := range rule.Filters {
					if filter.Type == gwv1alpha2.GRPCRouteFilterRequestMirror {
						if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableTLSService(service client.ObjectKey) bool {
	for key := range c.tlsroutes {
		// Get TLSRoute from client-go cache
		if r, err := c.getTLSRouteFromCache(key); err == nil {
			//r := r.(*gwv1alpha2.TLSRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableTCPService(service client.ObjectKey) bool {
	for key := range c.tcproutes {
		// Get TCPRoute from client-go cache
		if r, err := c.getTCPRouteFromCache(key); err == nil {
			//r := r.(*gwv1alpha2.TCPRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isEffectiveRoute(parentRefs []gwv1beta1.ParentReference) bool {
	if len(c.gateways) == 0 {
		return false
	}

	for _, parentRef := range parentRefs {
		for _, gw := range c.gateways {
			if gwutils.IsRefToGateway(parentRef, gw) {
				return true
			}
		}
	}

	return false
}

func (c *GatewayCache) isEffectiveTargetRef(targetRef gwv1alpha2.PolicyTargetReference) bool {
	if targetRef.Group != constants.GatewayAPIGroup {
		return false
	}

	if targetRef.Kind == constants.GatewayAPIGatewayKind {
		if len(c.gateways) == 0 {
			return false
		}

		for _, key := range c.gateways {
			gateway, err := c.getGatewayFromCache(key)
			if err != nil {
				log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
				continue
			}

			if gwutils.IsRefToTarget(targetRef, gateway) {
				return true
			}
		}
	}

	if targetRef.Kind == constants.GatewayAPIHTTPRouteKind {
		if len(c.httproutes) == 0 {
			return false
		}

		for key := range c.httproutes {
			route, err := c.getHTTPRouteFromCache(key)
			if err != nil {
				log.Error().Msgf("Failed to get HTTPRoute %s: %s", key, err)
				continue
			}

			if gwutils.IsRefToTarget(targetRef, route) {
				return true
			}
		}
	}

	if targetRef.Kind == constants.GatewayAPIGRPCRouteKind {
		if len(c.grpcroutes) == 0 {
			return false
		}

		for key := range c.grpcroutes {
			route, err := c.getGRPCRouteFromCache(key)
			if err != nil {
				log.Error().Msgf("Failed to get GRPCRoute %s: %s", key, err)
				continue
			}

			if gwutils.IsRefToTarget(targetRef, route) {
				return true
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableTargetService(owner client.Object, targetRef gwv1alpha2.PolicyTargetReference) bool {
	key := client.ObjectKey{
		Name:      string(targetRef.Name),
		Namespace: owner.GetNamespace(),
	}
	if targetRef.Namespace != nil {
		key.Namespace = string(*targetRef.Namespace)
	}

	if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
		(targetRef.Group == constants.FlomeshAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
		return c.isRoutableService(key)
	}

	return false
}

func (c *GatewayCache) isSecretReferred(secret client.ObjectKey) bool {
	//ctx := context.TODO()
	for _, key := range c.gateways {
		gw, err := c.getGatewayFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
			continue
		}

		for _, l := range gw.Spec.Listeners {
			switch l.Protocol {
			case gwv1beta1.HTTPSProtocolType, gwv1beta1.TLSProtocolType:
				if l.TLS == nil {
					continue
				}

				if l.TLS.Mode == nil || *l.TLS.Mode == gwv1beta1.TLSModeTerminate {
					if len(l.TLS.CertificateRefs) == 0 {
						continue
					}

					for _, ref := range l.TLS.CertificateRefs {
						if isRefToSecret(ref, secret, gw.Namespace) {
							return true
						}
					}
				}
			}
		}
	}

	for key := range c.upstreamstls {
		ut, err := c.getUpstreamTLSPolicyFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get UpstreamTLSPolicy %s: %s", key, err)
			continue
		}

		if ut.Spec.DefaultConfig != nil {
			if isRefToSecret(ut.Spec.DefaultConfig.CertificateRef, secret, ut.Namespace) {
				return true
			}
		}

		if len(ut.Spec.Ports) > 0 {
			for _, port := range ut.Spec.Ports {
				if port.Config == nil {
					continue
				}

				if isRefToSecret(port.Config.CertificateRef, secret, ut.Namespace) {
					return true
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) getGatewayFromCache(key client.ObjectKey) (*gwv1beta1.Gateway, error) {
	obj, err := c.informers.GetListers().Gateway.Gateways(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(gatewayGVK)

	return obj, nil
}

func (c *GatewayCache) getHTTPRouteFromCache(key client.ObjectKey) (*gwv1beta1.HTTPRoute, error) {
	obj, err := c.informers.GetListers().HTTPRoute.HTTPRoutes(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(httpRouteGVK)

	return obj, nil
}

func (c *GatewayCache) getGRPCRouteFromCache(key client.ObjectKey) (*gwv1alpha2.GRPCRoute, error) {
	obj, err := c.informers.GetListers().GRPCRoute.GRPCRoutes(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(grpcRouteGVK)

	return obj, nil
}

func (c *GatewayCache) getTLSRouteFromCache(key client.ObjectKey) (*gwv1alpha2.TLSRoute, error) {
	obj, err := c.informers.GetListers().TLSRoute.TLSRoutes(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(tlsRouteGVK)

	return obj, nil
}

func (c *GatewayCache) getTCPRouteFromCache(key client.ObjectKey) (*gwv1alpha2.TCPRoute, error) {
	obj, err := c.informers.GetListers().TCPRoute.TCPRoutes(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(tcpRouteGVK)

	return obj, nil
}

func (c *GatewayCache) getSecretFromCache(key client.ObjectKey) (*corev1.Secret, error) {
	obj, err := c.informers.GetListers().Secret.Secrets(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(secretGVK)

	return obj, nil
}

func (c *GatewayCache) getServiceFromCache(key client.ObjectKey) (*corev1.Service, error) {
	obj, err := c.informers.GetListers().Service.Services(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(serviceGVK)

	return obj, nil
}

func (c *GatewayCache) getRateLimitPolicyFromCache(key client.ObjectKey) (*gwpav1alpha1.RateLimitPolicy, error) {
	obj, err := c.informers.GetListers().RateLimitPolicy.RateLimitPolicies(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(rateLimitPolicyGVK)

	return obj, nil
}

func (c *GatewayCache) getSessionStickyPolicyFromCache(key client.ObjectKey) (*gwpav1alpha1.SessionStickyPolicy, error) {
	obj, err := c.informers.GetListers().SessionStickyPolicy.SessionStickyPolicies(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(sessionStickyPolicyGVK)

	return obj, nil
}

func (c *GatewayCache) getLoadBalancerPolicyFromCache(key client.ObjectKey) (*gwpav1alpha1.LoadBalancerPolicy, error) {
	obj, err := c.informers.GetListers().LoadBalancerPolicy.LoadBalancerPolicies(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(loadBalancerPolicyGVK)

	return obj, nil
}

func (c *GatewayCache) getCircuitBreakingPolicyFromCache(key client.ObjectKey) (*gwpav1alpha1.CircuitBreakingPolicy, error) {
	obj, err := c.informers.GetListers().CircuitBreakingPolicy.CircuitBreakingPolicies(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(circuitBreakingPolicyGVK)

	return obj, nil
}

func (c *GatewayCache) getAccessControlPolicyFromCache(key client.ObjectKey) (*gwpav1alpha1.AccessControlPolicy, error) {
	obj, err := c.informers.GetListers().AccessControlPolicy.AccessControlPolicies(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(accessControlPolicyGVK)

	return obj, nil
}

func (c *GatewayCache) getHealthCheckPolicyFromCache(key client.ObjectKey) (*gwpav1alpha1.HealthCheckPolicy, error) {
	obj, err := c.informers.GetListers().HealthCheckPolicy.HealthCheckPolicies(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(healthCheckPolicyGVK)

	return obj, nil
}

func (c *GatewayCache) getFaultInjectionPolicyFromCache(key client.ObjectKey) (*gwpav1alpha1.FaultInjectionPolicy, error) {
	obj, err := c.informers.GetListers().FaultInjectionPolicy.FaultInjectionPolicies(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(faultInjectionPolicyGVK)

	return obj, nil
}

func (c *GatewayCache) getUpstreamTLSPolicyFromCache(key client.ObjectKey) (*gwpav1alpha1.UpstreamTLSPolicy, error) {
	obj, err := c.informers.GetListers().UpstreamTLSPolicy.UpstreamTLSPolicies(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(upstreamTLSPolicyGVK)

	return obj, nil
}
