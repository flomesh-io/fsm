package provider

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	connectorv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
)

type MachineDiscoveryClient struct {
	machineClient      machineClientset.Interface
	deriveNamespace    string
	isInternalServices bool
	clusterId          string
}

func (dc *MachineDiscoveryClient) IsInternalServices() bool {
	return dc.isInternalServices
}

func (dc *MachineDiscoveryClient) CatalogServices(q *QueryOptions) (map[string][]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vms, err := dc.machineClient.MachineV1alpha1().VirtualMachines(dc.deriveNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	catalogServices := make(map[string][]string)
	if len(vms.Items) > 0 {
		for _, vm := range vms.Items {
			if len(vm.Spec.Services) == 0 {
				continue
			}
			for _, svc := range vm.Spec.Services {
				svcTagArray, exists := catalogServices[svc.ServiceName]
				if !exists {
					svcTagArray = make([]string, 0)
				}
				metadata := vm.Labels
				for k, v := range metadata {
					svcTagArray = append(svcTagArray, fmt.Sprintf("%s=%v", k, v))
				}
				catalogServices[svc.ServiceName] = svcTagArray
			}
		}
	}
	return catalogServices, nil
}

// HealthService is used to query catalog entries for a given service
func (dc *MachineDiscoveryClient) HealthService(service, _ string, _ *QueryOptions, _ bool) ([]*AgentService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vms, err := dc.machineClient.MachineV1alpha1().VirtualMachines(dc.deriveNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	agentServices := make([]*AgentService, 0)
	if len(vms.Items) > 0 {
		for _, vm := range vms.Items {
			updateVM := false
			if clusterId, exists := vm.Annotations[connector.AnnotationCloudServiceInheritedClusterID]; exists {
				if len(dc.clusterId) > 0 {
					if !strings.EqualFold(dc.clusterId, clusterId) {
						vm.Annotations[connector.AnnotationCloudServiceInheritedClusterID] = dc.clusterId
						updateVM = true
					}
				} else {
					delete(vm.Annotations, connector.AnnotationCloudServiceInheritedClusterID)
					updateVM = true
				}
			} else {
				if len(dc.clusterId) > 0 {
					vm.Annotations[connector.AnnotationCloudServiceInheritedClusterID] = dc.clusterId
					updateVM = true
				}
			}
			if _, internal := vm.Annotations[connector.AnnotationMeshServiceInternalSync]; internal {
				if !dc.isInternalServices {
					delete(vm.Annotations, connector.AnnotationMeshServiceInternalSync)
					updateVM = true
				}
			} else {
				if dc.isInternalServices {
					vm.Annotations[connector.AnnotationMeshServiceInternalSync] = "true"
				}
			}
			if updateVM {
				vm := vm
				if _, err = dc.machineClient.MachineV1alpha1().VirtualMachines(vm.Namespace).Update(ctx, &vm, metav1.UpdateOptions{}); err != nil {
					log.Error().Err(err)
					continue
				}
			}
			if len(vm.Spec.Services) == 0 {
				continue
			}
			for _, svc := range vm.Spec.Services {
				if strings.EqualFold(svc.ServiceName, service) {
					agentService := new(AgentService)
					agentService.fromVM(vm, svc)
					agentService.ClusterId = dc.clusterId
					agentServices = append(agentServices, agentService)
				}
			}
		}
	}
	return agentServices, nil
}

// CatalogService is used to query catalog entries for a given service
func (dc *MachineDiscoveryClient) CatalogService(service, tag string, q *QueryOptions) ([]*CatalogService, error) {
	// useless
	catalogServices := make([]*CatalogService, 0)
	return catalogServices, nil
}

func (dc *MachineDiscoveryClient) NodeServiceList(node string, q *QueryOptions) (*CatalogNodeServiceList, error) {
	// useless
	return nil, nil
}

func (dc *MachineDiscoveryClient) Deregister(dereg *CatalogDeregistration) error {
	// useless
	return nil
}

func (dc *MachineDiscoveryClient) Register(reg *CatalogRegistration) error {
	// useless
	return nil
}

// EnsureNamespaceExists ensures a Consul namespace with name ns exists. If it doesn't,
// it will create it and set crossNSACLPolicy as a policy default.
// Boolean return value indicates if the namespace was created by this call.
func (dc *MachineDiscoveryClient) EnsureNamespaceExists(ns string, crossNSAClPolicy string) (bool, error) {
	// useless
	return false, nil
}

func (dc *MachineDiscoveryClient) MicroServiceProvider() connectorv1alpha1.DiscoveryServiceProvider {
	return connectorv1alpha1.MachineDiscoveryService
}

func GetMachineDiscoveryClient(machineClient machineClientset.Interface, deriveNamespace string, isInternalServices bool, clusterId string) (*MachineDiscoveryClient, error) {
	machineDiscoveryClient := new(MachineDiscoveryClient)
	machineDiscoveryClient.machineClient = machineClient
	machineDiscoveryClient.deriveNamespace = deriveNamespace
	machineDiscoveryClient.isInternalServices = isInternalServices
	machineDiscoveryClient.clusterId = clusterId
	return machineDiscoveryClient, nil
}
