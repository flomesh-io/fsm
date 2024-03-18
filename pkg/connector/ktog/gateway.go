package ktog

import (
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
)

var (
	pathMatchType  = gwv1beta1.PathMatchPathPrefix
	pathMatchValue = "/"
)

// GatewaySource implements controller.Resource and starts
// a background watcher o gateway to keep track of changing gateway.
type GatewaySource struct {
	serviceResource  *KtoGSource
	gatewaysInformer cache.SharedIndexInformer
}

func (gw *GatewaySource) SetServiceResource(serviceResource *KtoGSource) {
	gw.serviceResource = serviceResource
}

func (gw *GatewaySource) Informer() cache.SharedIndexInformer {
	// Watch all k8s namespaces. Events will be filtered out as appropriate in the
	// `shouldTrackEndpoints` function which checks whether the Service is marked
	// to be tracked by the `shouldSync` function which uses the allow and deny
	// namespace lists.
	if gw.gatewaysInformer == nil {
		gw.gatewaysInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return gw.serviceResource.gatewayClient.GatewayV1beta1().Gateways(gw.serviceResource.fsmNamespace).List(gw.serviceResource.ctx, options)
				},

				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return gw.serviceResource.gatewayClient.GatewayV1beta1().Gateways(gw.serviceResource.fsmNamespace).Watch(gw.serviceResource.ctx, options)
				},
			},
			&gwv1beta1.Gateway{},
			0,
			cache.Indexers{},
		)
	}
	return gw.gatewaysInformer
}

func (gw *GatewaySource) Upsert(key string, raw interface{}) error {
	_, ok := raw.(*gwv1beta1.Gateway)
	if !ok {
		log.Warn().Msgf("Upsert got invalid type, raw:%v", raw)
		return nil
	}

	svcResource := gw.serviceResource
	svcResource.serviceLock.Lock()
	defer svcResource.serviceLock.Unlock()

	serviceList := svcResource.servicesInformer.GetIndexer().List()
	if len(serviceList) == 0 {
		return nil
	}
	for _, serviceEntry := range serviceList {
		service := serviceEntry.(*apiv1.Service)
		if !svcResource.shouldSync(service) {
			continue
		}
		gw.updateGatewayRoute(service)
	}

	log.Info().Msgf("upsert Gateway key:%s", key)
	return nil
}

func (gw *GatewaySource) Delete(key string, raw interface{}) error {
	return nil
}

func (gw *GatewaySource) updateGatewayRoute(k8sSvc *apiv1.Service) {
	_ = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		svcResource := gw.serviceResource
		gatewayList := gw.gatewaysInformer.GetIndexer().List()
		if len(gatewayList) == 0 {
			log.Warn().Msgf("error list gateways in namespace:%s", svcResource.fsmNamespace)
			return nil
		}
		for _, portSpec := range k8sSvc.Spec.Ports {
			protocol := string(portSpec.Protocol)
			if portSpec.AppProtocol != nil && len(*portSpec.AppProtocol) > 0 {
				protocol = *portSpec.AppProtocol
			}
			protocol = strings.ToUpper(protocol)

			internalSource := true
			if len(k8sSvc.Annotations) > 0 {
				if _, externalSource := k8sSvc.Annotations[connector.AnnotationMeshServiceSync]; externalSource {
					_, internalSource = k8sSvc.Annotations[connector.AnnotationMeshServiceInternalSync]
				}
			}

			var parentRefs []gwv1beta1.ParentReference
			for _, gatewayEntry := range gatewayList {
				gateway := gatewayEntry.(*gwv1beta1.Gateway)
				for _, gatewayListener := range gateway.Spec.Listeners {
					//glProtocol := strings.ToUpper(string(gatewayListener.Protocol))
					//glName := strings.ToUpper(string(gatewayListener.Name))
					if internalSource {
						if httpPort := gw.serviceResource.controller.GetViaIngressHTTPPort(); httpPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) &&
							uint(gatewayListener.Port) == httpPort {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
						if grpcPort := gw.serviceResource.controller.GetViaIngressGRPCPort(); grpcPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) &&
							uint(gatewayListener.Port) == grpcPort {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
					} else {
						if httpPort := gw.serviceResource.controller.GetViaEgressHTTPPort(); httpPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) &&
							uint(gatewayListener.Port) == httpPort {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
						if grpcPort := gw.serviceResource.controller.GetViaEgressGRPCPort(); grpcPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) &&
							uint(gatewayListener.Port) == grpcPort {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
					}
				}
			}

			if len(parentRefs) > 0 {
				if strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) {
					gw.updateGatewayHTTPRoute(k8sSvc, portSpec, parentRefs)
				} else if strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) {
					gw.updateGatewayGRPCRoute(k8sSvc, portSpec, parentRefs)
				} else {
					gw.updateGatewayTCPRoute(k8sSvc, portSpec, parentRefs)
				}
			} else {
				log.Warn().Msgf("error match gateways in namespace:%s for svc:%s/%s protocol:%s", svcResource.fsmNamespace, k8sSvc.Namespace, k8sSvc.Name, protocol)
			}
		}
		return nil
	})
}

func (gw *GatewaySource) updateGatewayHTTPRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) {
	var newRt *gwv1beta1.HTTPRoute
	var exists bool
	svcResource := gw.serviceResource
	httpRouteClient := svcResource.gatewayClient.GatewayV1beta1().HTTPRoutes(k8sSvc.Namespace)
	if existRt, err := httpRouteClient.Get(svcResource.ctx, k8sSvc.Name, metav1.GetOptions{}); err != nil {
		newRt = &gwv1beta1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: k8sSvc.Namespace,
				Name:      k8sSvc.Name,
			},
		}
	} else {
		newRt = existRt.DeepCopy()
		exists = true
	}
	weight := int32(constants.ClusterWeightAcceptAll)
	servicePort := gwv1beta1.PortNumber(portSpec.Port)
	newRt.Spec.CommonRouteSpec.ParentRefs = parentRefs
	newRt.Spec.Hostnames = gw.getGatewayRouteHostnamesForService(k8sSvc)
	newRt.Spec.Rules = []gwv1beta1.HTTPRouteRule{{
		Matches: []gwv1beta1.HTTPRouteMatch{{
			Path: &gwv1beta1.HTTPPathMatch{
				Type:  &pathMatchType,
				Value: &pathMatchValue,
			},
		}},
		BackendRefs: []gwv1beta1.HTTPBackendRef{
			{
				BackendRef: gwv1beta1.BackendRef{
					BackendObjectReference: gwv1beta1.BackendObjectReference{
						Name: gwv1beta1.ObjectName(k8sSvc.Name),
						Port: &servicePort,
					},
					Weight: &weight,
				},
			},
		},
	}}

	if !exists {
		_, err := httpRouteClient.Create(svcResource.ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating http route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		_, err := httpRouteClient.Update(svcResource.ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating http route, name:%s warn:%v", k8sSvc.Name, err)
		}
	}
}

func (gw *GatewaySource) updateGatewayGRPCRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) {
	var newRt *gwv1alpha2.GRPCRoute
	var exists bool
	svcResource := gw.serviceResource
	grpcRouteClient := svcResource.gatewayClient.GatewayV1alpha2().GRPCRoutes(k8sSvc.Namespace)
	if existRt, err := grpcRouteClient.Get(svcResource.ctx, k8sSvc.Name, metav1.GetOptions{}); err != nil {
		newRt = &gwv1alpha2.GRPCRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: k8sSvc.Namespace,
				Name:      k8sSvc.Name,
			},
		}
	} else {
		newRt = existRt.DeepCopy()
		exists = true
	}
	grpMethodType := gwv1alpha2.GRPCMethodMatchExact
	grpMethodSvc := "grpc.GrpcService"
	servicePort := gwv1beta1.PortNumber(portSpec.Port)
	weight := int32(constants.ClusterWeightAcceptAll)
	newRt.Spec.CommonRouteSpec.ParentRefs = parentRefs
	newRt.Spec.Hostnames = gw.getGatewayRouteHostnamesForService(k8sSvc)
	newRt.Spec.Rules = []gwv1alpha2.GRPCRouteRule{{
		Matches: []gwv1alpha2.GRPCRouteMatch{{
			Method: &gwv1alpha2.GRPCMethodMatch{
				Type:    &grpMethodType,
				Service: &grpMethodSvc,
			},
		}},
		BackendRefs: []gwv1alpha2.GRPCBackendRef{
			{
				BackendRef: gwv1beta1.BackendRef{
					BackendObjectReference: gwv1beta1.BackendObjectReference{
						Name: gwv1beta1.ObjectName(k8sSvc.Name),
						Port: &servicePort,
					},
					Weight: &weight,
				},
			},
		},
	}}

	if !exists {
		_, err := grpcRouteClient.Create(svcResource.ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating grpc route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		_, err := grpcRouteClient.Update(svcResource.ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating grpc route, name:%s warn:%v", k8sSvc.Name, err)
		}
	}
}

func (gw *GatewaySource) updateGatewayTCPRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) {
	var newRt *gwv1alpha2.TCPRoute
	var exists bool
	svcResource := gw.serviceResource
	tcpRouteClient := svcResource.gatewayClient.GatewayV1alpha2().TCPRoutes(k8sSvc.Namespace)
	if existRt, err := tcpRouteClient.Get(svcResource.ctx, k8sSvc.Name, metav1.GetOptions{}); err != nil {
		newRt = &gwv1alpha2.TCPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: k8sSvc.Namespace,
				Name:      k8sSvc.Name,
			},
		}
	} else {
		newRt = existRt.DeepCopy()
		exists = true
	}
	servicePort := gwv1beta1.PortNumber(portSpec.Port)
	weight := int32(constants.ClusterWeightAcceptAll)
	newRt.Spec.CommonRouteSpec.ParentRefs = parentRefs
	newRt.Spec.Rules = []gwv1alpha2.TCPRouteRule{{
		BackendRefs: []gwv1alpha2.BackendRef{
			{
				BackendObjectReference: gwv1beta1.BackendObjectReference{
					Name: gwv1beta1.ObjectName(k8sSvc.Name),
					Port: &servicePort,
				},
				Weight: &weight,
			},
		},
	}}

	if !exists {
		_, err := tcpRouteClient.Create(svcResource.ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating tcp route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		_, err := tcpRouteClient.Update(svcResource.ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating tcp route, name:%s warn:%v", k8sSvc.Name, err)
		}
	}
}

func (gw *GatewaySource) deleteGatewayRoute(name, namespace string) {
	svcResource := gw.serviceResource
	httpRouteClient := svcResource.gatewayClient.GatewayV1beta1().HTTPRoutes(namespace)
	_ = httpRouteClient.Delete(svcResource.ctx, name, metav1.DeleteOptions{})

	grpcRouteClient := svcResource.gatewayClient.GatewayV1alpha2().GRPCRoutes(namespace)
	_ = grpcRouteClient.Delete(svcResource.ctx, name, metav1.DeleteOptions{})

	tcpRouteClient := svcResource.gatewayClient.GatewayV1alpha2().TCPRoutes(namespace)
	_ = tcpRouteClient.Delete(svcResource.ctx, name, metav1.DeleteOptions{})
}

func (gw *GatewaySource) getGatewayRouteHostnamesForService(k8sSvc *apiv1.Service) []gwv1beta1.Hostname {
	svcResource := gw.serviceResource
	hostnames := []gwv1beta1.Hostname{
		gwv1beta1.Hostname(k8sSvc.Name),
		gwv1beta1.Hostname(fmt.Sprintf("%s.%s", k8sSvc.Name, k8sSvc.Namespace)),
		gwv1beta1.Hostname(fmt.Sprintf("%s.%s.svc", k8sSvc.Name, k8sSvc.Namespace)),
	}
	endpointsClient := svcResource.kubeClient.CoreV1().Endpoints(k8sSvc.Namespace)
	if endpoints, err := endpointsClient.Get(svcResource.ctx, k8sSvc.Name, metav1.GetOptions{}); err == nil {
		for _, subsets := range endpoints.Subsets {
			for _, addr := range subsets.Addresses {
				hostnames = append(hostnames, gwv1beta1.Hostname(addr.IP))
			}
		}
	}
	return hostnames
}
