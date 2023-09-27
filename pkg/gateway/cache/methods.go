package cache

import (
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// Insert inserts an object into the cache
func (c *GatewayCache) Insert(obj interface{}) bool {
	p := c.getProcessor(obj)
	if p != nil {
		return p.Insert(obj, c)
	}

	return false
}

// Delete deletes an object from the cache
func (c *GatewayCache) Delete(obj interface{}) bool {
	p := c.getProcessor(obj)
	if p != nil {
		return p.Delete(obj, c)
	}

	return false
}

//func (c *GatewayCache) WaitForCacheSync(ctx context.Context) bool {
//	return c.cache.WaitForCacheSync(ctx)
//}

func (c *GatewayCache) getProcessor(obj interface{}) Processor {
	switch obj.(type) {
	case *corev1.Service:
		return c.processors[ServicesProcessorType]
	case *mcsv1alpha1.ServiceImport:
		return c.processors[ServiceImportsProcessorType]
	//case *corev1.Endpoints:
	//	return c.processors[EndpointsProcessorType]
	case *discoveryv1.EndpointSlice:
		return c.processors[EndpointSlicesProcessorType]
	case *corev1.Secret:
		return c.processors[SecretsProcessorType]
	case *gwv1beta1.GatewayClass:
		return c.processors[GatewayClassesProcessorType]
	case *gwv1beta1.Gateway:
		return c.processors[GatewaysProcessorType]
	case *gwv1beta1.HTTPRoute:
		return c.processors[HTTPRoutesProcessorType]
	case *gwv1alpha2.GRPCRoute:
		return c.processors[GRPCRoutesProcessorType]
	case *gwv1alpha2.TCPRoute:
		return c.processors[TCPRoutesProcessorType]
	case *gwv1alpha2.TLSRoute:
		return c.processors[TLSRoutesProcessorType]
	}

	return nil
}

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
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIHTTPRoute, key.String()); exists && err == nil {
			r := r.(*gwv1beta1.HTTPRoute)
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
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIGRPCRoute, key.String()); exists && err == nil {
			r := r.(*gwv1alpha2.GRPCRoute)
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
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPITLSRoute, key.String()); exists && err == nil {
			r := r.(*gwv1alpha2.TLSRoute)
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
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPITCPRoute, key.String()); exists && err == nil {
			r := r.(*gwv1alpha2.TCPRoute)
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

func (c *GatewayCache) isSecretReferredByAnyGateway(secret client.ObjectKey) bool {
	//ctx := context.TODO()
	for _, key := range c.gateways {
		//gw := &gwv1beta1.Gateway{}
		//if err := c.cache.Get(ctx, key, gw); err != nil {
		//	log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
		//	continue
		//}
		obj, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIGateway, key.String())
		if err != nil {
			log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
			continue
		}
		if !exists {
			log.Error().Msgf("Gateway %s doesn't exist", key)
			continue
		}

		gw := obj.(*gwv1beta1.Gateway)

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

	return false
}
