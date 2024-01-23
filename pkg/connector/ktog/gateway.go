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

// GatewayResource implements controller.Resource and starts
// a background watcher o gateway to keep track of changing gateway.
type GatewayResource struct {
	Service          *ServiceResource
	gatewaysInformer cache.SharedIndexInformer
}

func (gw *GatewayResource) Informer() cache.SharedIndexInformer {
	// Watch all k8s namespaces. Events will be filtered out as appropriate in the
	// `shouldTrackEndpoints` function which checks whether the Service is marked
	// to be tracked by the `shouldSync` function which uses the allow and deny
	// namespace lists.
	if gw.gatewaysInformer == nil {
		gw.gatewaysInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return gw.Service.GatewayClient.GatewayV1beta1().Gateways(gw.Service.FsmNamespace).List(gw.Service.Ctx, options)
				},

				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return gw.Service.GatewayClient.GatewayV1beta1().Gateways(gw.Service.FsmNamespace).Watch(gw.Service.Ctx, options)
				},
			},
			&gwv1beta1.Gateway{},
			0,
			cache.Indexers{},
		)
	}
	return gw.gatewaysInformer
}

func (gw *GatewayResource) Upsert(key string, raw interface{}) error {
	_, ok := raw.(*gwv1beta1.Gateway)
	if !ok {
		log.Warn().Msgf("Upsert got invalid type, raw:%v", raw)
		return nil
	}

	svc := gw.Service
	svc.serviceLock.Lock()
	defer svc.serviceLock.Unlock()

	serviceList := svc.servicesInformer.GetIndexer().List()
	if len(serviceList) == 0 {
		return nil
	}
	for _, serviceEntry := range serviceList {
		service := serviceEntry.(*apiv1.Service)
		if !svc.shouldSync(service) {
			continue
		}
		gw.updateGatewayRoute(service)
	}

	log.Info().Msgf("upsert Gateway key:%s", key)
	return nil
}

func (gw *GatewayResource) Delete(key string, raw interface{}) error {
	return nil
}

func (gw *GatewayResource) updateGatewayRoute(k8sSvc *apiv1.Service) {
	_ = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		svc := gw.Service
		gatewayList := gw.gatewaysInformer.GetIndexer().List()
		if len(gatewayList) == 0 {
			log.Warn().Msgf("error list gateways in namespace:%s", svc.FsmNamespace)
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
						if connector.ViaGateway.Ingress.HTTPPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) &&
							uint(gatewayListener.Port) == connector.ViaGateway.Ingress.HTTPPort {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
						if connector.ViaGateway.Ingress.GRPCPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) &&
							uint(gatewayListener.Port) == connector.ViaGateway.Ingress.GRPCPort {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
					} else {
						if connector.ViaGateway.Egress.HTTPPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) &&
							uint(gatewayListener.Port) == connector.ViaGateway.Egress.HTTPPort {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
						if connector.ViaGateway.Egress.GRPCPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) &&
							uint(gatewayListener.Port) == connector.ViaGateway.Egress.GRPCPort {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
					}
					//if strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) {
					//	if strings.EqualFold(glProtocol, strings.ToUpper(constants.ProtocolHTTP)) &&
					//		strings.HasPrefix(glName, protocol) {
					//		gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
					//		gatewayPort := gatewayListener.HTTPPort
					//		parentRefs = append(parentRefs, gwv1beta1.ParentReference{
					//			Namespace: &gatewayNs,
					//			Name:      gwv1beta1.ObjectName(gateway.Name),
					//			HTTPPort:      &gatewayPort})
					//	}
					//} else if strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) {
					//	if strings.EqualFold(glProtocol, protocol) &&
					//		!strings.HasPrefix(glName, strings.ToUpper(constants.ProtocolGRPC)) {
					//		gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
					//		gatewayPort := gatewayListener.HTTPPort
					//		parentRefs = append(parentRefs, gwv1beta1.ParentReference{
					//			Namespace: &gatewayNs,
					//			Name:      gwv1beta1.ObjectName(gateway.Name),
					//			HTTPPort:      &gatewayPort})
					//	}
					//}
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
				log.Warn().Msgf("error match gateways in namespace:%s for svc:%s/%s protocol:%s", svc.FsmNamespace, k8sSvc.Namespace, k8sSvc.Name, protocol)
			}
		}
		return nil
	})
}

func (gw *GatewayResource) updateGatewayHTTPRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) {
	var newRt *gwv1beta1.HTTPRoute
	var exists bool
	svc := gw.Service
	httpRouteClient := svc.GatewayClient.GatewayV1beta1().HTTPRoutes(k8sSvc.Namespace)
	if existRt, err := httpRouteClient.Get(svc.Ctx, k8sSvc.Name, metav1.GetOptions{}); err != nil {
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
		_, err := httpRouteClient.Create(svc.Ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating http route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		_, err := httpRouteClient.Update(svc.Ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating http route, name:%s warn:%v", k8sSvc.Name, err)
		}
	}
}

func (gw *GatewayResource) updateGatewayGRPCRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) {
	var newRt *gwv1alpha2.GRPCRoute
	var exists bool
	svc := gw.Service
	grpcRouteClient := svc.GatewayClient.GatewayV1alpha2().GRPCRoutes(k8sSvc.Namespace)
	if existRt, err := grpcRouteClient.Get(svc.Ctx, k8sSvc.Name, metav1.GetOptions{}); err != nil {
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
		_, err := grpcRouteClient.Create(svc.Ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating grpc route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		_, err := grpcRouteClient.Update(svc.Ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating grpc route, name:%s warn:%v", k8sSvc.Name, err)
		}
	}
}

func (gw *GatewayResource) updateGatewayTCPRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) {
	var newRt *gwv1alpha2.TCPRoute
	var exists bool
	svc := gw.Service
	tcpRouteClient := svc.GatewayClient.GatewayV1alpha2().TCPRoutes(k8sSvc.Namespace)
	if existRt, err := tcpRouteClient.Get(svc.Ctx, k8sSvc.Name, metav1.GetOptions{}); err != nil {
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
		_, err := tcpRouteClient.Create(svc.Ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating tcp route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		_, err := tcpRouteClient.Update(svc.Ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating tcp route, name:%s warn:%v", k8sSvc.Name, err)
		}
	}
}

func (gw *GatewayResource) deleteGatewayRoute(name, namespace string) {
	svc := gw.Service
	httpRouteClient := svc.GatewayClient.GatewayV1beta1().HTTPRoutes(namespace)
	_ = httpRouteClient.Delete(svc.Ctx, name, metav1.DeleteOptions{})

	grpcRouteClient := svc.GatewayClient.GatewayV1alpha2().GRPCRoutes(namespace)
	_ = grpcRouteClient.Delete(svc.Ctx, name, metav1.DeleteOptions{})

	tcpRouteClient := svc.GatewayClient.GatewayV1alpha2().TCPRoutes(namespace)
	_ = tcpRouteClient.Delete(svc.Ctx, name, metav1.DeleteOptions{})
}

func (gw *GatewayResource) getGatewayRouteHostnamesForService(k8sSvc *apiv1.Service) []gwv1beta1.Hostname {
	svc := gw.Service
	hostnames := []gwv1beta1.Hostname{
		gwv1beta1.Hostname(k8sSvc.Name),
		gwv1beta1.Hostname(fmt.Sprintf("%s.%s", k8sSvc.Name, k8sSvc.Namespace)),
		gwv1beta1.Hostname(fmt.Sprintf("%s.%s.svc", k8sSvc.Name, k8sSvc.Namespace)),
	}
	endpointsClient := svc.Client.CoreV1().Endpoints(k8sSvc.Namespace)
	if endpoints, err := endpointsClient.Get(svc.Ctx, k8sSvc.Name, metav1.GetOptions{}); err == nil {
		for _, subsets := range endpoints.Subsets {
			for _, addr := range subsets.Addresses {
				hostnames = append(hostnames, gwv1beta1.Hostname(addr.IP))
			}
		}
	}
	return hostnames
}
