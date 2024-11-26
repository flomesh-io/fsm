package ktog

import (
	"fmt"
	"strings"

	"github.com/mitchellh/hashstructure/v2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
)

var (
	pathMatchType  = gwv1.PathMatchPathPrefix
	pathMatchValue = "/"
)

var (
	gatewayGroup = gwv1.Group(gwv1.GroupName)
	gatewayKind  = gwv1.Kind("Gateway")
	serviceGroup = gwv1.Group("")
	serviceKind  = gwv1.Kind("Service")
)

// GatewaySource implements controller.Resource and starts
// a background watcher o gateway to keep track of changing gateway.
type GatewaySource struct {
	serviceResource  *KtoGSource
	gatewaysInformer cache.SharedIndexInformer
	informers        *fsminformers.InformerCollection
	InterceptionMode string
}

func (gw *GatewaySource) SetServiceResource(serviceResource *KtoGSource) {
	gw.serviceResource = serviceResource
}

func (gw *GatewaySource) SetInformers(informers *fsminformers.InformerCollection) {
	gw.informers = informers
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
					return gw.serviceResource.gatewayClient.GatewayV1().Gateways(gw.serviceResource.fsmNamespace).List(gw.serviceResource.ctx, options)
				},

				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return gw.serviceResource.gatewayClient.GatewayV1().Gateways(gw.serviceResource.fsmNamespace).Watch(gw.serviceResource.ctx, options)
				},
			},
			&gwv1.Gateway{},
			0,
			cache.Indexers{},
		)
	}
	return gw.gatewaysInformer
}

func (gw *GatewaySource) Upsert(key string, raw interface{}) error {
	_, ok := raw.(*gwv1.Gateway)
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

func (gw *GatewaySource) Delete(_ string, _ interface{}) error {
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
		externalSource := false
		internalSource := true
		if len(k8sSvc.Annotations) > 0 {
			internalSource, externalSource = gw.checkServiceType(k8sSvc)
		}
		for _, portSpec := range k8sSvc.Spec.Ports {
			protocol := string(portSpec.Protocol)
			if portSpec.AppProtocol != nil && len(*portSpec.AppProtocol) > 0 {
				protocol = *portSpec.AppProtocol
			}
			protocol = strings.ToUpper(protocol)

			var parentRefs []gwv1.ParentReference
			for _, gatewayEntry := range gatewayList {
				gateway := gatewayEntry.(*gwv1.Gateway)
				for _, gatewayListener := range gateway.Spec.Listeners {
					if internalSource {
						if httpPort := gw.serviceResource.controller.GetViaIngressHTTPPort(); httpPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) &&
							uint(gatewayListener.Port) == httpPort {
							gatewayNs := gwv1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1.ParentReference{
								Group:     &gatewayGroup,
								Kind:      &gatewayKind,
								Namespace: &gatewayNs,
								Name:      gwv1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
						if grpcPort := gw.serviceResource.controller.GetViaIngressGRPCPort(); grpcPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) &&
							uint(gatewayListener.Port) == grpcPort {
							gatewayNs := gwv1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1.ParentReference{
								Group:     &gatewayGroup,
								Kind:      &gatewayKind,
								Namespace: &gatewayNs,
								Name:      gwv1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
					}
					if externalSource ||
						(internalSource && gw.InterceptionMode == constants.TrafficInterceptionModeNodeLevel) {
						if httpPort := gw.serviceResource.controller.GetViaEgressHTTPPort(); httpPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) &&
							uint(gatewayListener.Port) == httpPort {
							gatewayNs := gwv1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1.ParentReference{
								Group:     &gatewayGroup,
								Kind:      &gatewayKind,
								Namespace: &gatewayNs,
								Name:      gwv1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
						if grpcPort := gw.serviceResource.controller.GetViaEgressGRPCPort(); grpcPort > 0 &&
							strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) &&
							uint(gatewayListener.Port) == grpcPort {
							gatewayNs := gwv1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1.ParentReference{
								Group:     &gatewayGroup,
								Kind:      &gatewayKind,
								Namespace: &gatewayNs,
								Name:      gwv1.ObjectName(gateway.Name),
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

func (gw *GatewaySource) checkServiceType(k8sSvc *apiv1.Service) (internalSource, externalSource bool) {
	externalSource = false
	internalSource = true
	if v, exists := k8sSvc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
		svcMeta := connector.Decode(k8sSvc, v)
		for _, endpointMeta := range svcMeta.Endpoints {
			if !endpointMeta.Local.WithGateway {
				continue
			}
			if !internalSource && endpointMeta.Local.InternalService {
				internalSource = true
			}

			if !externalSource && !endpointMeta.Local.InternalService {
				if endpointMeta.Local.WithMultiGateways {
					externalSource = true
				}
			}
			if internalSource && externalSource {
				break
			}
		}
	} else {
		internalSource = true
	}
	return internalSource, externalSource
}

func (gw *GatewaySource) updateGatewayHTTPRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1.ParentReference) {
	svcResource := gw.serviceResource
	httpRouteClient := svcResource.gatewayClient.GatewayV1().HTTPRoutes(k8sSvc.Namespace)
	existRt := gw.GetHTTPRoute(k8sSvc.Name, k8sSvc.Namespace)

	weight := int32(constants.ClusterWeightAcceptAll)
	servicePort := gwv1.PortNumber(portSpec.Port)

	newRt := &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: k8sSvc.Namespace,
			Name:      k8sSvc.Name,
		},
	}
	newRt.Spec.CommonRouteSpec.ParentRefs = parentRefs
	newRt.Spec.Hostnames = gw.getGatewayRouteHostnamesForService(k8sSvc)
	newRt.Spec.Rules = []gwv1.HTTPRouteRule{{
		Matches: []gwv1.HTTPRouteMatch{{
			Path: &gwv1.HTTPPathMatch{
				Type:  &pathMatchType,
				Value: &pathMatchValue,
			},
		}},
		BackendRefs: []gwv1.HTTPBackendRef{
			{
				BackendRef: gwv1.BackendRef{
					BackendObjectReference: gwv1.BackendObjectReference{
						Name:  gwv1.ObjectName(k8sSvc.Name),
						Port:  &servicePort,
						Group: &serviceGroup,
						Kind:  &serviceKind,
					},
					Weight: &weight,
				},
			},
		},
	}}

	if existRt == nil {
		if _, err := httpRouteClient.Create(svcResource.ctx, newRt, metav1.CreateOptions{}); err != nil {
			log.Error().Msgf("warn creating http route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		existRtHash, _ := hashstructure.Hash(existRt.Spec, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			})
		newRtHash, _ := hashstructure.Hash(newRt.Spec, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			})
		if existRtHash != newRtHash {
			existRt.Spec = newRt.Spec
			if _, err := httpRouteClient.Update(svcResource.ctx, existRt, metav1.UpdateOptions{}); err != nil {
				log.Error().Msgf("warn updating http route, name:%s warn:%v", k8sSvc.Name, err)
			}
		}
	}
}

func (gw *GatewaySource) updateGatewayGRPCRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1.ParentReference) {
	svcResource := gw.serviceResource
	grpcRouteClient := svcResource.gatewayClient.GatewayV1().GRPCRoutes(k8sSvc.Namespace)
	existRt := gw.GetGRPCRoute(k8sSvc.Name, k8sSvc.Namespace)

	grpMethodType := gwv1.GRPCMethodMatchExact
	grpMethodSvc := "grpc.GrpcService"
	servicePort := gwv1.PortNumber(portSpec.Port)
	weight := int32(constants.ClusterWeightAcceptAll)

	newRt := &gwv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: k8sSvc.Namespace,
			Name:      k8sSvc.Name,
		},
	}
	newRt.Spec.CommonRouteSpec.ParentRefs = parentRefs
	newRt.Spec.Hostnames = gw.getGatewayRouteHostnamesForService(k8sSvc)
	newRt.Spec.Rules = []gwv1.GRPCRouteRule{{
		Matches: []gwv1.GRPCRouteMatch{{
			Method: &gwv1.GRPCMethodMatch{
				Type:    &grpMethodType,
				Service: &grpMethodSvc,
			},
		}},
		BackendRefs: []gwv1.GRPCBackendRef{
			{
				BackendRef: gwv1.BackendRef{
					BackendObjectReference: gwv1.BackendObjectReference{
						Name:  gwv1.ObjectName(k8sSvc.Name),
						Port:  &servicePort,
						Group: &serviceGroup,
						Kind:  &serviceKind,
					},
					Weight: &weight,
				},
			},
		},
	}}

	if existRt == nil {
		if _, err := grpcRouteClient.Create(svcResource.ctx, newRt, metav1.CreateOptions{}); err != nil {
			log.Error().Msgf("warn creating grpc route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		existRtHash, _ := hashstructure.Hash(existRt.Spec, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			})
		newRtHash, _ := hashstructure.Hash(newRt.Spec, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			})

		if existRtHash != newRtHash {
			existRt.Spec = newRt.Spec
			if _, err := grpcRouteClient.Update(svcResource.ctx, newRt, metav1.UpdateOptions{}); err != nil {
				log.Error().Msgf("warn updating grpc route, name:%s warn:%v", k8sSvc.Name, err)
			}
		}
	}
}

func (gw *GatewaySource) updateGatewayTCPRoute(k8sSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1.ParentReference) {
	svcResource := gw.serviceResource
	tcpRouteClient := svcResource.gatewayClient.GatewayV1alpha2().TCPRoutes(k8sSvc.Namespace)
	existRt := gw.GetTCPRoute(k8sSvc.Name, k8sSvc.Namespace)

	servicePort := gwv1.PortNumber(portSpec.Port)
	weight := int32(constants.ClusterWeightAcceptAll)

	newRt := &gwv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: k8sSvc.Namespace,
			Name:      k8sSvc.Name,
		},
	}
	newRt.Spec.CommonRouteSpec.ParentRefs = parentRefs
	newRt.Spec.Rules = []gwv1alpha2.TCPRouteRule{{
		BackendRefs: []gwv1alpha2.BackendRef{
			{
				BackendObjectReference: gwv1.BackendObjectReference{
					Name:  gwv1.ObjectName(k8sSvc.Name),
					Port:  &servicePort,
					Group: &serviceGroup,
					Kind:  &serviceKind,
				},
				Weight: &weight,
			},
		},
	}}

	if existRt == nil {
		if _, err := tcpRouteClient.Create(svcResource.ctx, newRt, metav1.CreateOptions{}); err != nil {
			log.Error().Msgf("warn creating tcp route, name:%s warn:%v", k8sSvc.Name, err)
		}
	} else {
		existRtHash, _ := hashstructure.Hash(existRt.Spec, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			})
		newRtHash, _ := hashstructure.Hash(newRt.Spec, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			})

		if existRtHash != newRtHash {
			existRt.Spec = newRt.Spec
			if _, err := tcpRouteClient.Update(svcResource.ctx, newRt, metav1.UpdateOptions{}); err != nil {
				log.Error().Msgf("warn updating tcp route, name:%s warn:%v", k8sSvc.Name, err)
			}
		}
	}
}

func (gw *GatewaySource) deleteGatewayRoute(name, namespace string) {
	svcResource := gw.serviceResource
	if routeIf := gw.GetHTTPRoute(name, namespace); routeIf != nil {
		httpRouteClient := svcResource.gatewayClient.GatewayV1().HTTPRoutes(namespace)
		_ = httpRouteClient.Delete(svcResource.ctx, name, metav1.DeleteOptions{})
	}

	if routeIf := gw.GetGRPCRoute(name, namespace); routeIf != nil {
		grpcRouteClient := svcResource.gatewayClient.GatewayV1().GRPCRoutes(namespace)
		_ = grpcRouteClient.Delete(svcResource.ctx, name, metav1.DeleteOptions{})
	}

	if routeIf := gw.GetTCPRoute(name, namespace); routeIf != nil {
		tcpRouteClient := svcResource.gatewayClient.GatewayV1alpha2().TCPRoutes(namespace)
		_ = tcpRouteClient.Delete(svcResource.ctx, name, metav1.DeleteOptions{})
	}
}

func (gw *GatewaySource) getGatewayRouteHostnamesForService(k8sSvc *apiv1.Service) []gwv1.Hostname {
	svcResource := gw.serviceResource
	hostnames := []gwv1.Hostname{
		gwv1.Hostname(k8sSvc.Name),
		gwv1.Hostname(fmt.Sprintf("%s.%s", k8sSvc.Name, k8sSvc.Namespace)),
		gwv1.Hostname(fmt.Sprintf("%s.%s.svc", k8sSvc.Name, k8sSvc.Namespace)),
	}
	cloudService := false
	if len(k8sSvc.Annotations) > 0 {
		if v, exists := k8sSvc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
			cloudService = true
			svcMeta := connector.Decode(k8sSvc, v)
			for addr := range svcMeta.Endpoints {
				hostnames = append(hostnames, gwv1.Hostname(addr))
			}
		}
	}
	if !cloudService {
		endpointsClient := svcResource.kubeClient.CoreV1().Endpoints(k8sSvc.Namespace)
		if endpoints, err := endpointsClient.Get(svcResource.ctx, k8sSvc.Name, metav1.GetOptions{}); err == nil {
			for _, subsets := range endpoints.Subsets {
				for _, addr := range subsets.Addresses {
					hostnames = append(hostnames, gwv1.Hostname(addr.IP))
				}
			}
		}
	}
	return hostnames
}

// GetHTTPRoute returns a HTTPRoute resource if found, nil otherwise.
func (gw *GatewaySource) GetHTTPRoute(route, namespace string) *gwv1.HTTPRoute {
	routeIf, exists, err := gw.informers.GetByKey(fsminformers.InformerKeyGatewayAPIHTTPRoute, fmt.Sprintf("%s/%s", namespace, route))
	if exists && err == nil {
		return routeIf.(*gwv1.HTTPRoute)
	}
	return nil
}

// GetGRPCRoute returns a GRPCRoute resource if found, nil otherwise.
func (gw *GatewaySource) GetGRPCRoute(route, namespace string) *gwv1.GRPCRoute {
	routeIf, exists, err := gw.informers.GetByKey(fsminformers.InformerKeyGatewayAPIGRPCRoute, fmt.Sprintf("%s/%s", namespace, route))
	if exists && err == nil {
		return routeIf.(*gwv1.GRPCRoute)
	}
	return nil
}

// GetTCPRoute returns a TCPRoute resource if found, nil otherwise.
func (gw *GatewaySource) GetTCPRoute(route, namespace string) *gwv1alpha2.TCPRoute {
	routeIf, exists, err := gw.informers.GetByKey(fsminformers.InformerKeyGatewayAPITCPRoute, fmt.Sprintf("%s/%s", namespace, route))
	if exists && err == nil {
		return routeIf.(*gwv1alpha2.TCPRoute)
	}
	return nil
}
