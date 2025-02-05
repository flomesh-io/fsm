// Package kube implements KubeClient's methods.
package kube

import (
	"net"
	"strings"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/service"
)

// Ensure interface compliance
var _ endpoint.Provider = (*client)(nil)
var _ service.Provider = (*client)(nil)

// NewClient returns a client that has all components necessary to connect to and maintain state of a Kubernetes cluster.
func NewClient(kubeController k8s.Controller, cfg configurator.Configurator) *client { //nolint: revive // unexported-return
	return &client{
		kubeController:   kubeController,
		meshConfigurator: cfg,
	}
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interfaces: EndpointsProvider, ServiceProvider
func (c *client) GetID() string {
	return providerName
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c *client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	log.Trace().Msgf("Getting Endpoints for MeshService %s on Kubernetes", svc)

	var endpoints []endpoint.Endpoint

	k8sSvc := c.kubeController.GetService(svc)
	if k8sSvc != nil && len(k8sSvc.Annotations) > 0 {
		if v, exists := k8sSvc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
			svcMeta := connector.Decode(k8sSvc, v)
			if len(svcMeta.Endpoints) > 0 {
				lbType := c.meshConfigurator.GetMeshConfig().Spec.Connector.Lb.Type
				for addr, endpointMeta := range svcMeta.Endpoints {
					for port, protocol := range endpointMeta.Ports {
						ept := endpoint.Endpoint{
							IP:                net.ParseIP(string(addr)),
							Port:              endpoint.Port(port),
							AppProtocol:       string(protocol),
							ClusterID:         endpointMeta.Native.ClusterId,
							ViaGatewayHTTP:    endpointMeta.Native.ViaGatewayHTTP,
							ViaGatewayGRPC:    endpointMeta.Native.ViaGatewayGRPC,
							ViaGatewayMode:    string(endpointMeta.Native.ViaGatewayMode),
							WithGateway:       endpointMeta.Local.WithGateway,
							WithMultiGateways: endpointMeta.Local.WithMultiGateways,
						}
						if !endpointMeta.Local.InternalService {
							ept.ClusterKey = endpointMeta.Native.ClusterSet
							ept.LBType = string(lbType)
						}
						endpoints = append(endpoints, ept)
					}
				}
			}
			return endpoints
		}
	}

	kubernetesEndpoints, err := c.kubeController.GetEndpoints(svc)
	if err != nil || kubernetesEndpoints == nil {
		log.Info().Msgf("No k8s endpoints found for MeshService %s", svc)
		return nil
	}

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
				ept := endpoint.Endpoint{
					IP:   ip,
					Port: endpoint.Port(port.Port),
				}
				if port.AppProtocol != nil {
					ept.AppProtocol = *port.AppProtocol
				} else if len(port.Name) > 0 {
					if strings.Contains(port.Name, constants.ProtocolHTTP) {
						ept.AppProtocol = constants.ProtocolHTTP
					} else if strings.Contains(port.Name, constants.ProtocolGRPC) {
						ept.AppProtocol = constants.ProtocolGRPC
					}
				}
				endpoints = append(endpoints, ept)
			}
		}
	}

	log.Trace().Msgf("Endpoints for MeshService %s: %v", svc, endpoints)

	return endpoints
}

// ListEndpointsForIdentity retrieves the list of IP addresses for the given service account
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/flomesh-io/fsm/issues/3188]
func (c *client) ListEndpointsForIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	sa := serviceIdentity.ToK8sServiceAccount()
	log.Trace().Msgf("[%s] (ListEndpointsForIdentity) Getting Endpoints for service account %s on Kubernetes", c.GetID(), sa)

	var endpoints []endpoint.Endpoint

	for _, pod := range c.kubeController.ListPods() {
		if pod.Namespace != sa.Namespace {
			continue
		}
		if pod.Spec.ServiceAccountName != sa.Name {
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

	for _, vm := range c.kubeController.ListVms() {
		if vm.Namespace != sa.Namespace {
			continue
		}
		if vm.Spec.ServiceAccountName != sa.Name {
			continue
		}

		ip := net.ParseIP(vm.Spec.MachineIP)
		if ip == nil {
			log.Error().Msgf("[%s] Error parsing IP address %s", c.GetID(), vm.Spec.MachineIP)
			break
		}
		ept := endpoint.Endpoint{IP: ip}
		endpoints = append(endpoints, ept)
	}

	log.Trace().Msgf("[%s][ListEndpointsForIdentity] Endpoints for service identity (serviceAccount=%s) %s: %+v", c.GetID(), serviceIdentity, sa, endpoints)

	return endpoints
}

// GetServicesForServiceIdentity retrieves a list of services for the given service identity.
func (c *client) GetServicesForServiceIdentity(svcIdentity identity.ServiceIdentity) []service.MeshService {
	var meshServices []service.MeshService
	svcSet := mapset.NewSet() // mapset is used to avoid duplicate elements in the output list

	svcAccount := svcIdentity.ToK8sServiceAccount()

	for _, pod := range c.kubeController.ListPods() {
		if pod.Namespace != svcAccount.Namespace {
			continue
		}

		if pod.Spec.ServiceAccountName != svcAccount.Name {
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

	for _, vm := range c.kubeController.ListVms() {
		if vm.Namespace != svcAccount.Namespace {
			continue
		}

		if vm.Spec.ServiceAccountName != svcAccount.Name {
			continue
		}

		podLabels := vm.ObjectMeta.Labels
		meshServicesForPod := c.getServicesByLabels(podLabels, vm.Namespace)
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
	serviceList := c.kubeController.ListServices(true, true)

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
			finalList = append(finalList, k8s.ServiceToMeshServices(c.kubeController, svc)...)
		}
	}

	return finalList
}

// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service
// FQDN is resolved
func (c *client) GetResolvableEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint

	// Check if the service has been given Cluster IP
	kubeService := c.kubeController.GetService(svc)
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

	var ips []net.IP
	ips = append(ips, ip)

	sam := c.meshConfigurator.GetServiceAccessMode()
	if sam == configv1alpha3.ServiceAccessModeIP || sam == configv1alpha3.ServiceAccessModeMixed {
		if eps, err := c.kubeController.GetEndpoints(svc); err == nil && eps != nil {
			if len(eps.Subsets) > 0 {
				for _, ep := range eps.Subsets {
					if len(ep.Addresses) > 0 {
						for _, addr := range ep.Addresses {
							ips = append(ips, net.ParseIP(addr.IP))
						}
					}
				}
			}
		}
	}

	for _, svcPort := range kubeService.Spec.Ports {
		for _, addr := range ips {
			endpoints = append(endpoints, endpoint.Endpoint{
				IP:   addr,
				Port: endpoint.Port(svcPort.Port),
			})
		}
	}

	return endpoints
}

// ListServices returns a list of services that are part of monitored namespaces
func (c *client) ListServices() []service.MeshService {
	var services []service.MeshService
	for _, svc := range c.kubeController.ListServices(true, true) {
		if len(svc.Annotations) > 0 {
			if _, exists := svc.Annotations[connector.AnnotationCloudHealthCheckService]; exists {
				continue
			}
		}
		services = append(services, k8s.ServiceToMeshServices(c.kubeController, svc)...)
	}
	return services
}

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (c *client) ListServiceIdentitiesForService(svc service.MeshService) []identity.ServiceIdentity {
	serviceAccounts, err := c.kubeController.ListServiceIdentitiesForService(svc)
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
