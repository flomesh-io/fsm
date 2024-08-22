package provider

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
)

type MachineDiscoveryClient struct {
	connectController connector.ConnectController
	machineClient     machineClientset.Interface
}

func (dc *MachineDiscoveryClient) IsInternalServices() bool {
	return dc.connectController.AsInternalServices()
}

func (dc *MachineDiscoveryClient) CatalogInstances(service string, _ *connector.QueryOptions) ([]*connector.AgentService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vms, err := dc.machineClient.MachineV1alpha1().VirtualMachines(dc.connectController.GetDeriveNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	agentServices := make([]*connector.AgentService, 0)
	if len(vms.Items) > 0 {
		for _, vm := range vms.Items {
			updateVM := false
			if clusterId, exists := vm.Annotations[connector.AnnotationCloudServiceInheritedClusterID]; exists {
				if len(dc.connectController.GetClusterId()) > 0 {
					if !strings.EqualFold(dc.connectController.GetClusterId(), clusterId) {
						vm.Annotations[connector.AnnotationCloudServiceInheritedClusterID] = dc.connectController.GetClusterId()
						updateVM = true
					}
				} else {
					delete(vm.Annotations, connector.AnnotationCloudServiceInheritedClusterID)
					updateVM = true
				}
			} else {
				if len(dc.connectController.GetClusterId()) > 0 {
					vm.Annotations[connector.AnnotationCloudServiceInheritedClusterID] = dc.connectController.GetClusterId()
					updateVM = true
				}
			}
			if _, internal := vm.Annotations[connector.AnnotationMeshServiceInternalSync]; internal {
				if !dc.connectController.AsInternalServices() {
					delete(vm.Annotations, connector.AnnotationMeshServiceInternalSync)
					updateVM = true
				}
			} else {
				if dc.connectController.AsInternalServices() {
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
			if filterIPRanges := dc.connectController.GetC2KFilterIPRanges(); len(filterIPRanges) > 0 {
				include := false
				for _, cidr := range filterIPRanges {
					if cidr.Contains(vm.Spec.MachineIP) {
						include = true
						break
					}
				}
				if !include {
					continue
				}
			}
			if excludeIPRanges := dc.connectController.GetC2KExcludeIPRanges(); len(excludeIPRanges) > 0 {
				exclude := false
				for _, cidr := range excludeIPRanges {
					if cidr.Contains(vm.Spec.MachineIP) {
						exclude = true
						break
					}
				}
				if exclude {
					continue
				}
			}
			for _, svc := range vm.Spec.Services {
				if strings.EqualFold(svc.ServiceName, service) {
					agentService := new(connector.AgentService)
					agentService.FromVM(vm, svc)
					agentService.ClusterId = dc.connectController.GetClusterId()
					agentServices = append(agentServices, agentService)
				}
			}
		}
	}
	return agentServices, nil
}

func (dc *MachineDiscoveryClient) CatalogServices(*connector.QueryOptions) ([]connector.MicroService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vms, err := dc.machineClient.MachineV1alpha1().VirtualMachines(dc.connectController.GetDeriveNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var catalogServices []connector.MicroService
	if len(vms.Items) > 0 {
		for _, vm := range vms.Items {
			if len(vm.Spec.Services) == 0 {
				continue
			}
			if filterIPRanges := dc.connectController.GetC2KFilterIPRanges(); len(filterIPRanges) > 0 {
				include := false
				for _, cidr := range filterIPRanges {
					if cidr.Contains(vm.Spec.MachineIP) {
						include = true
						break
					}
				}
				if !include {
					continue
				}
			}
			if excludeIPRanges := dc.connectController.GetC2KExcludeIPRanges(); len(excludeIPRanges) > 0 {
				exclude := false
				for _, cidr := range excludeIPRanges {
					if cidr.Contains(vm.Spec.MachineIP) {
						exclude = true
						break
					}
				}
				if exclude {
					continue
				}
			}
			for _, svc := range vm.Spec.Services {
				catalogServices = append(catalogServices, connector.MicroService{Service: svc.ServiceName})
			}
		}
	}
	return catalogServices, nil
}

// RegisteredInstances is used to query catalog entries for a given service
func (dc *MachineDiscoveryClient) RegisteredInstances(string, *connector.QueryOptions) ([]*connector.CatalogService, error) {
	// useless
	catalogServices := make([]*connector.CatalogService, 0)
	return catalogServices, nil
}

func (dc *MachineDiscoveryClient) RegisteredServices(*connector.QueryOptions) ([]connector.MicroService, error) {
	// useless
	return nil, nil
}

func (dc *MachineDiscoveryClient) Deregister(*connector.CatalogDeregistration) error {
	// useless
	return nil
}

func (dc *MachineDiscoveryClient) Register(*connector.CatalogRegistration) error {
	// useless
	return nil
}

func (dc *MachineDiscoveryClient) EnableNamespaces() bool {
	return false
}

// EnsureNamespaceExists ensures a namespace with name ns exists.
func (dc *MachineDiscoveryClient) EnsureNamespaceExists(string) (bool, error) {
	// useless
	return false, nil
}

// RegisteredNamespace returns the cloud namespace that a service should be
// registered in based on the namespace options. It returns an
// empty string if namespaces aren't enabled.
func (dc *MachineDiscoveryClient) RegisteredNamespace(string) string {
	return ""
}

func (dc *MachineDiscoveryClient) MicroServiceProvider() ctv1.DiscoveryServiceProvider {
	return ctv1.MachineDiscoveryService
}

func (dc *MachineDiscoveryClient) Close() {
}

func GetMachineDiscoveryClient(connectController connector.ConnectController,
	machineClient machineClientset.Interface) (*MachineDiscoveryClient, error) {
	machineDiscoveryClient := new(MachineDiscoveryClient)
	machineDiscoveryClient.connectController = connectController
	machineDiscoveryClient.machineClient = machineClient
	return machineDiscoveryClient, nil
}
