package c2k

import (
	"context"
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/constants"
)

var (
	pathMatchType  = gwv1beta1.PathMatchPathPrefix
	pathMatchValue = "/"
)

// UpsertGateway implements the controller.Resource interface.
func (s *Sink) UpsertGateway(key string, raw interface{}) error {
	// We expect a Gateway. If it isn't a Gateway then just ignore it.
	_, ok := raw.(*gwv1beta1.Gateway)
	if !ok {
		log.Warn().Msgf("UpsertGateway got invalid type, raw:%v", raw)
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	serviceList := s.servicesInformer.GetIndexer().List()
	if len(serviceList) == 0 {
		return nil
	}
	for _, serviceEntry := range serviceList {
		service := serviceEntry.(*apiv1.Service)
		s.updateGatewayRoute(service)
	}

	log.Trace().Msgf("UpsertGateway, key:%s", key)
	return nil
}

// DeleteGateway implements the controller.Resource interface.
func (s *Sink) DeleteGateway(_ string, _ interface{}) error {
	return nil
}

func (s *Sink) updateGatewayRoute(createdSvc *apiv1.Service) {
	_ = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		gatewayList := s.gatewaysInformer.GetIndexer().List()
		if len(gatewayList) == 0 {
			log.Warn().Msgf("error list gateways in namespace:%s", s.fsmNamespace)
			return nil
		}
		for _, portSpec := range createdSvc.Spec.Ports {
			protocol := *portSpec.AppProtocol
			if len(protocol) == 0 {
				protocol = string(portSpec.Protocol)
			}
			protocol = strings.ToUpper(protocol)

			var parentRefs []gwv1beta1.ParentReference

			for _, gatewayEntry := range gatewayList {
				gateway := gatewayEntry.(*gwv1beta1.Gateway)
				for _, gatewayListener := range gateway.Spec.Listeners {
					glProtocol := strings.ToUpper(string(gatewayListener.Protocol))
					glName := strings.ToUpper(string(gatewayListener.Name))
					if strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) {
						if strings.EqualFold(glProtocol, strings.ToUpper(constants.ProtocolHTTP)) &&
							strings.HasPrefix(glName, protocol) {
							gatewayNs := gwv1beta1.Namespace(gateway.Namespace)
							gatewayPort := gatewayListener.Port
							parentRefs = append(parentRefs, gwv1beta1.ParentReference{
								Namespace: &gatewayNs,
								Name:      gwv1beta1.ObjectName(gateway.Name),
								Port:      &gatewayPort})
						}
					} else if strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolHTTP)) {
						if strings.EqualFold(glProtocol, protocol) &&
							!strings.HasPrefix(glName, strings.ToUpper(constants.ProtocolGRPC)) {
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
					if err := s.updateGatewayHTTPRoute(createdSvc, portSpec, parentRefs); err != nil {
						return err
					}
				} else if strings.EqualFold(protocol, strings.ToUpper(constants.ProtocolGRPC)) {
					if err := s.updateGatewayGRPCRoute(createdSvc, portSpec, parentRefs); err != nil {
						return err
					}
				} else {
					if err := s.updateGatewayTCPRoute(createdSvc, portSpec, parentRefs); err != nil {
						return err
					}
				}
			} else {
				log.Warn().Msgf("error match gateways in namespace:%s", s.fsmNamespace)
			}
		}
		return nil
	})
}

func (s *Sink) updateGatewayHTTPRoute(createdSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) error {
	var newRt *gwv1beta1.HTTPRoute
	var exists bool
	httpRouteClient := s.gatewayClient.GatewayV1beta1().HTTPRoutes(createdSvc.Namespace)
	if existRt, err := httpRouteClient.Get(s.Ctx, createdSvc.Name, metav1.GetOptions{}); err != nil {
		newRt = &gwv1beta1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: s.namespace(),
				Name:      createdSvc.Name,
			},
		}
	} else {
		newRt = existRt.DeepCopy()
		exists = true
	}
	weight := int32(constants.ClusterWeightAcceptAll)
	servicePort := gwv1beta1.PortNumber(portSpec.Port)
	newRt.Spec.CommonRouteSpec.ParentRefs = parentRefs
	newRt.Spec.Hostnames = s.getGatewayRouteHostnamesForService(createdSvc)
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
						Name: gwv1beta1.ObjectName(createdSvc.Name),
						Port: &servicePort,
					},
					Weight: &weight,
				},
			},
		},
	}}

	if !exists {
		_, err := httpRouteClient.Create(s.Ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating http route, name:%s warn:%v", createdSvc.Name, err)
		}
		return err
	} else {
		_, err := httpRouteClient.Update(s.Ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating http route, name:%s warn:%v", createdSvc.Name, err)
		}
		return err
	}
}

func (s *Sink) updateGatewayGRPCRoute(createdSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) error {
	var newRt *gwv1alpha2.GRPCRoute
	var exists bool
	grpcRouteClient := s.gatewayClient.GatewayV1alpha2().GRPCRoutes(createdSvc.Namespace)
	if existRt, err := grpcRouteClient.Get(s.Ctx, createdSvc.Name, metav1.GetOptions{}); err != nil {
		newRt = &gwv1alpha2.GRPCRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: s.namespace(),
				Name:      createdSvc.Name,
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
	newRt.Spec.Hostnames = s.getGatewayRouteHostnamesForService(createdSvc)
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
						Name: gwv1beta1.ObjectName(createdSvc.Name),
						Port: &servicePort,
					},
					Weight: &weight,
				},
			},
		},
	}}

	if !exists {
		_, err := grpcRouteClient.Create(s.Ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating grpc route, name:%s warn:%v", createdSvc.Name, err)
		}
		return err
	} else {
		_, err := grpcRouteClient.Update(s.Ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating grpc route, name:%s warn:%v", createdSvc.Name, err)
		}
		return err
	}
}

func (s *Sink) updateGatewayTCPRoute(createdSvc *apiv1.Service, portSpec apiv1.ServicePort, parentRefs []gwv1beta1.ParentReference) error {
	var newRt *gwv1alpha2.TCPRoute
	var exists bool
	tcpRouteClient := s.gatewayClient.GatewayV1alpha2().TCPRoutes(createdSvc.Namespace)
	if existRt, err := tcpRouteClient.Get(s.Ctx, createdSvc.Name, metav1.GetOptions{}); err != nil {
		newRt = &gwv1alpha2.TCPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: s.namespace(),
				Name:      createdSvc.Name,
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
					Name: gwv1beta1.ObjectName(createdSvc.Name),
					Port: &servicePort,
				},
				Weight: &weight,
			},
		},
	}}

	if !exists {
		_, err := tcpRouteClient.Create(s.Ctx, newRt, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("warn creating tcp route, name:%s warn:%v", createdSvc.Name, err)
		}
		return err
	} else {
		_, err := tcpRouteClient.Update(s.Ctx, newRt, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("warn updating tcp route, name:%s warn:%v", createdSvc.Name, err)
		}
		return err
	}
}

func (s *Sink) deleteGatewayRoute(name string) {
	httpRouteClient := s.gatewayClient.GatewayV1beta1().HTTPRoutes(s.namespace())
	_ = httpRouteClient.Delete(s.Ctx, name, metav1.DeleteOptions{})

	grpcRouteClient := s.gatewayClient.GatewayV1alpha2().GRPCRoutes(s.namespace())
	_ = grpcRouteClient.Delete(s.Ctx, name, metav1.DeleteOptions{})

	tcpRouteClient := s.gatewayClient.GatewayV1alpha2().TCPRoutes(s.namespace())
	_ = tcpRouteClient.Delete(s.Ctx, name, metav1.DeleteOptions{})
}

func (s *Sink) updateGatewayEndpointSlice(ctx context.Context, endpoints *apiv1.Endpoints) {
	endpointsDup := endpoints.DeepCopy()
	_ = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		eptSliceClient := s.KubeClient.DiscoveryV1().EndpointSlices(endpointsDup.Namespace)
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
			constants.KubernetesEndpointSliceServiceNameLabel: endpointsDup.Name,
		}}
		listOptions := metav1.ListOptions{LabelSelector: labels.Set(labelSelector.MatchLabels).String()}
		eptSliceList, err := eptSliceClient.List(ctx, listOptions)
		if err != nil {
			return err
		}
		if eptSliceList == nil || len(eptSliceList.Items) == 0 {
			return fmt.Errorf("not exists EndpointSlice, name:%s", endpointsDup.Name)
		}
		newEpSlice := eptSliceList.Items[0].DeepCopy()
		if len(newEpSlice.Labels) > 0 {
			delete(newEpSlice.Labels, "endpointslice.kubernetes.io/managed-by")
		}
		var ports []discoveryv1.EndpointPort
		var epts []discoveryv1.Endpoint
		for _, subsets := range endpointsDup.Subsets {
			for _, port := range subsets.Ports {
				shadow := port
				ports = append(ports, discoveryv1.EndpointPort{
					Name:        &shadow.Name,
					Protocol:    &shadow.Protocol,
					Port:        &shadow.Port,
					AppProtocol: shadow.AppProtocol,
				})
			}
			if len(subsets.Addresses) > 0 {
				var ready = true
				var addrs []string
				ept := discoveryv1.Endpoint{
					Conditions: discoveryv1.EndpointConditions{
						Ready: &ready,
					},
				}
				for _, addr := range subsets.Addresses {
					addrs = append(addrs, addr.IP)
				}
				ept.Addresses = addrs
				epts = append(epts, ept)
			}
		}
		newEpSlice.Ports = ports
		newEpSlice.Endpoints = epts

		_, err = eptSliceClient.Update(s.Ctx, newEpSlice, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("error updating EndpointSlice, name:%s warn:%v", newEpSlice.Name, err)
		}
		return err
	})
}

func (s *Sink) getGatewayRouteHostnamesForService(createdSvc *apiv1.Service) []gwv1beta1.Hostname {
	hostnames := []gwv1beta1.Hostname{gwv1beta1.Hostname(createdSvc.Name)}
	endpointsClient := s.KubeClient.CoreV1().Endpoints(createdSvc.Namespace)
	if endpoints, err := endpointsClient.Get(s.Ctx, createdSvc.Name, metav1.GetOptions{}); err == nil {
		for _, subsets := range endpoints.Subsets {
			for _, addr := range subsets.Addresses {
				hostnames = append(hostnames, gwv1beta1.Hostname(addr.IP))
			}
		}
	}
	return hostnames
}
