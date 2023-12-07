package provider

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
)

type MachineDiscoveryClient struct {
	machineClient      machineClientset.Interface
	isInternalServices bool
}

func (dc *MachineDiscoveryClient) IsInternalServices() bool {
	return dc.isInternalServices
}

func (dc *MachineDiscoveryClient) CatalogServices(q *QueryOptions) (map[string][]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vms, err := dc.machineClient.MachineV1alpha1().VirtualMachines(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	catalogServices := make(map[string][]string)
	if len(vms.Items) > 0 {
		for _, vm := range vms.Items {
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
	vms, err := dc.machineClient.MachineV1alpha1().VirtualMachines(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	agentServices := make([]*AgentService, 0)
	if len(vms.Items) > 0 {
		for _, vm := range vms.Items {
			for _, svc := range vm.Spec.Services {
				if strings.EqualFold(svc.ServiceName, service) {
					agentService := new(AgentService)
					agentService.fromVM(vm, svc)
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

func (dc *MachineDiscoveryClient) MicroServiceProvider() string {
	return connector.MachineDiscoveryService
}

func GetMachineDiscoveryClient(machineClient machineClientset.Interface, isInternalServices bool) (*MachineDiscoveryClient, error) {
	machineDiscoveryClient := new(MachineDiscoveryClient)
	machineDiscoveryClient.machineClient = machineClient
	machineDiscoveryClient.isInternalServices = isInternalServices
	return machineDiscoveryClient, nil
}
