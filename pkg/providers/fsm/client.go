// Package fsm implements MulticlusterClient's methods.
package fsm

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/multicluster"
	"github.com/flomesh-io/fsm/pkg/service"
)

// Ensure interface compliance
var _ endpoint.Provider = (*client)(nil)
var _ service.Provider = (*client)(nil)

// NewClient returns a client that has all components necessary to connect to and maintain state of a multi cluster.
func NewClient(multiclusterController multicluster.Controller, cfg configurator.Configurator) *client { //nolint: revive // unexported-return
	return &client{
		multiclusterController: multiclusterController,
		meshConfigurator:       cfg,
	}
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interfaces: EndpointsProvider, ServiceProvider
func (c *client) GetID() string {
	return providerName
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c *client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	kubernetesEndpoints, err := c.multiclusterController.GetEndpoints(svc)
	if err != nil || kubernetesEndpoints == nil {
		log.Info().Msgf("No mcs endpoints found for MeshService %s", svc)
		return nil
	}

	var endpoints []endpoint.Endpoint
	for _, kubernetesEndpoint := range kubernetesEndpoints.Subsets {
		for _, port := range kubernetesEndpoint.Ports {
			// If a TargetPort is specified for the service, filter the endpoint by this port.
			// This is required to ensure we do not attempt to filter the endpoints when the endpoints
			// are being listed for a MeshService whose TargetPort is not known.
			if svc.TargetPort != 0 && port.Port != int32(svc.TargetPort) {
				// k8s service's port does not match MeshService port, ignore this port
				continue
			}
			for _, address := range kubernetesEndpoint.Addresses {
				if svc.Subdomain() != "" && svc.Subdomain() != address.Hostname {
					// if there's a subdomain on this meshservice, make sure it matches the endpoint's hostname
					continue
				}
				ip := net.ParseIP(address.IP)
				if ip == nil {
					log.Error().Msgf("Error parsing endpoint IP address %s for MeshService %s", address.IP, svc)
					continue
				}
				weight, _ := strconv.ParseUint(kubernetesEndpoints.Annotations[fmt.Sprintf(multicluster.ServiceImportLBWeightAnnotation, address.IP, port.Port)], 10, 32)
				ept := endpoint.Endpoint{
					IP:         ip,
					Port:       endpoint.Port(port.Port),
					ClusterKey: kubernetesEndpoints.Annotations[fmt.Sprintf(multicluster.ServiceImportClusterKeyAnnotation, address.IP, port.Port)],
					LBType:     kubernetesEndpoints.Annotations[fmt.Sprintf(multicluster.ServiceImportLBTypeAnnotation, address.IP, port.Port)],
					Weight:     endpoint.Weight(weight),
					Path:       kubernetesEndpoints.Annotations[fmt.Sprintf(multicluster.ServiceImportContextPathAnnotation, address.IP, port.Port)],
				}
				endpoints = append(endpoints, ept)
			}
		}
	}
	return endpoints
}

// ListEndpointsForIdentity retrieves the list of IP addresses for the given service account
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/flomesh-io/fsm/issues/3188]
func (c *client) ListEndpointsForIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	sa := serviceIdentity.ToK8sServiceAccount()
	var endpoints []endpoint.Endpoint
	for _, pod := range c.multiclusterController.ListPods() {
		if pod.Namespace != sa.Namespace {
			continue
		}
		if pod.Spec.ServiceAccountName != sa.Name &&
			pod.Spec.ServiceAccountName != multicluster.AnyServiceAccount {
			continue
		}

		for _, podIP := range pod.Status.PodIPs {
			ip := net.ParseIP(podIP.IP)
			if ip == nil {
				log.Error().Msgf("[%s] Error parsing IP address %s", c.GetID(), podIP.IP)
				break
			}
			ept := endpoint.Endpoint{IP: ip}
			endpoints = append(endpoints, ept)
		}
	}
	return endpoints
}

// GetServicesForServiceIdentity retrieves a list of services for the given service identity.
func (c *client) GetServicesForServiceIdentity(svcIdentity identity.ServiceIdentity) []service.MeshService {
	var meshServices []service.MeshService
	svcSet := mapset.NewSet() // mapset is used to avoid duplicate elements in the output list

	svcAccount := svcIdentity.ToK8sServiceAccount()

	for _, pod := range c.multiclusterController.ListPods() {
		if pod.Namespace != svcAccount.Namespace {
			continue
		}

		if pod.Spec.ServiceAccountName != svcAccount.Name &&
			pod.Spec.ServiceAccountName != multicluster.AnyServiceAccount {
			continue
		}

		podLabels := pod.ObjectMeta.Labels
		meshServicesForPod := c.getServicesByLabels(podLabels, pod.Namespace)
		for _, svc := range meshServicesForPod {
			if added := svcSet.Add(svc); added {
				meshServices = append(meshServices, svc)
			}
		}
	}

	log.Trace().Msgf("[%s] Services for service account %s: %v", c.GetID(), svcAccount, meshServices)
	return meshServices
}

// getServicesByLabels gets Kubernetes services whose selectors match the given labels
func (c *client) getServicesByLabels(podLabels map[string]string, targetNamespace string) []service.MeshService {
	var finalList []service.MeshService
	serviceList := c.multiclusterController.ListServices()

	for _, svc := range serviceList {
		// TODO: #1684 Introduce APIs to dynamically allow applying selectors, instead of callers implementing
		// filtering themselves
		if svc.Namespace != targetNamespace {
			continue
		}

		svcRawSelector := svc.Spec.Selector
		// service has no selectors, we do not need to match against the pod label
		if len(svcRawSelector) == 0 {
			continue
		}
		selector := labels.Set(svcRawSelector).AsSelector()
		if selector.Matches(labels.Set(podLabels)) {
			finalList = append(finalList, ServiceToMeshServices(c.multiclusterController, *svc)...)
		}
	}

	return finalList
}

// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service
// FQDN is resolved
func (c *client) GetResolvableEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint

	// Check if the service has been given Cluster IP
	kubeService := c.multiclusterController.GetService(svc)
	if kubeService == nil {
		log.Info().Msgf("No k8s services found for MeshService %s", svc)
		return nil
	}

	if len(kubeService.Spec.ClusterIP) == 0 || kubeService.Spec.ClusterIP == corev1.ClusterIPNone {
		// If service has no cluster IP or cluster IP is <none>, use final endpoint as resolvable destinations
		return c.ListEndpointsForService(svc)
	}

	// Cluster IP is present
	ip := net.ParseIP(kubeService.Spec.ClusterIP)
	if ip == nil {
		log.Error().Msgf("[%s] Could not parse Cluster IP %s", c.GetID(), kubeService.Spec.ClusterIP)
		return nil
	}

	for _, svcPort := range kubeService.Spec.Ports {
		endpoints = append(endpoints, endpoint.Endpoint{
			IP:         ip,
			Port:       endpoint.Port(svcPort.Port),
			ClusterKey: c.GetID(),
		})
	}

	return endpoints
}

// ListServices returns a list of services that are part of monitored namespaces
func (c *client) ListServices() []service.MeshService {
	var services []service.MeshService
	for _, svc := range c.multiclusterController.ListServices() {
		services = append(services, ServiceToMeshServices(c.multiclusterController, *svc)...)
	}
	return services
}

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (c *client) ListServiceIdentitiesForService(svc service.MeshService) []identity.ServiceIdentity {
	serviceAccounts, err := c.multiclusterController.ListServiceIdentitiesForService(svc)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting ServiceAccounts for Service %s", svc)
		return nil
	}

	var serviceIdentities []identity.ServiceIdentity
	for _, svcAccount := range serviceAccounts {
		serviceIdentity := svcAccount.ToServiceIdentity()
		serviceIdentities = append(serviceIdentities, serviceIdentity)
	}

	return serviceIdentities
}

// ServiceToMeshServices translates a k8s service with one or more ports to one or more
// MeshService objects per port.
func ServiceToMeshServices(c multicluster.Controller, svc corev1.Service) []service.MeshService {
	var meshServices []service.MeshService

	for _, portSpec := range svc.Spec.Ports {
		meshSvc := service.MeshService{
			Namespace:        svc.Namespace,
			Name:             svc.Name,
			Port:             uint16(portSpec.Port),
			ServiceImportUID: svc.UID,
		}

		// attempt to parse protocol from port name
		// Order of Preference is:
		// 1. port.appProtocol field
		// 2. protocol prefixed to port name (e.g. tcp-my-port)
		// 3. default to http
		protocol := constants.ProtocolHTTP
		for _, p := range constants.SupportedProtocolsInMesh {
			if strings.HasPrefix(portSpec.Name, p+"-") {
				protocol = p
				break
			}
		}

		// use port.appProtocol if specified, else use port protocol
		meshSvc.Protocol = pointer.StringDeref(portSpec.AppProtocol, protocol)

		// The endpoints for the kubernetes service carry information that allows
		// us to retrieve the TargetPort for the MeshService.
		endpoints, _ := c.GetEndpoints(meshSvc)
		if endpoints == nil {
			continue
		}

		for _, subset := range endpoints.Subsets {
			for _, port := range subset.Ports {
				meshServices = append(meshServices, service.MeshService{
					Namespace:        meshSvc.Namespace,
					Name:             meshSvc.Name,
					Port:             meshSvc.Port,
					TargetPort:       uint16(port.Port),
					Protocol:         meshSvc.Protocol,
					ServiceImportUID: meshSvc.ServiceImportUID,
				})
			}
		}
	}

	return meshServices
}
