package registry

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/ctok"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/service"
	pipy2 "github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
)

// ProxyServiceMapper knows how to map Sidecar instances to services.
type ProxyServiceMapper interface {
	ListProxyServices(*pipy2.Proxy) ([]service.MeshService, error)
}

// ExplicitProxyServiceMapper is a custom ProxyServiceMapper implementation.
type ExplicitProxyServiceMapper func(*pipy2.Proxy) ([]service.MeshService, error)

// ListProxyServices executes the given mapping.
func (e ExplicitProxyServiceMapper) ListProxyServices(p *pipy2.Proxy) ([]service.MeshService, error) {
	return e(p)
}

// KubeProxyServiceMapper maps an Sidecar instance to services in a Kubernetes cluster.
type KubeProxyServiceMapper struct {
	KubeController k8s.Controller
}

// ListProxyServices maps an Pipy instance to a number of Kubernetes services.
func (k *KubeProxyServiceMapper) ListProxyServices(p *pipy2.Proxy) ([]service.MeshService, error) {
	var meshServices []service.MeshService
	if p.VM {
		vm, err := k.KubeController.GetVmForProxy(p)
		if err != nil {
			return nil, err
		}

		p.MachineIP = pipy2.NewNetAddress(vm.Spec.MachineIP)
		if _, internal := vm.Annotations[connector.AnnotationMeshServiceInternalSync]; internal {
			p.ClusterID = ""
		} else {
			if clusterId, exists := vm.Annotations[connector.AnnotationCloudServiceInheritedClusterID]; exists {
				p.ClusterID = clusterId
			}
		}

		meshServices = listServicesForVm(vm, k.KubeController)

		servicesForPod := strings.Join(listServiceNames(meshServices), ",")
		log.Trace().Msgf("Services associated with VM with UID=%s Name=%s/%s: %+v",
			vm.ObjectMeta.UID, vm.Namespace, vm.Name, servicesForPod)
	} else {
		pod, err := k.KubeController.GetPodForProxy(p)
		if err != nil {
			return nil, err
		}

		meshServices = listServicesForPod(pod, k.KubeController)

		servicesForPod := strings.Join(listServiceNames(meshServices), ",")
		log.Trace().Msgf("Services associated with Pod with UID=%s Name=%s/%s: %+v",
			pod.ObjectMeta.UID, pod.Namespace, pod.Name, servicesForPod)
	}

	return meshServices, nil
}

func kubernetesServicesToMeshServices(kubeController k8s.Controller, kubernetesServices []v1.Service, subdomainFilter string) (meshServices []service.MeshService) {
	for _, svc := range kubernetesServices {
		svc := svc
		for _, meshSvc := range k8s.ServiceToMeshServices(kubeController, &svc) {
			if meshSvc.Subdomain() == subdomainFilter || meshSvc.Subdomain() == "" {
				meshServices = append(meshServices, meshSvc)
			}
		}
	}
	return meshServices
}

func listServiceNames(meshServices []service.MeshService) (serviceNames []string) {
	for _, meshService := range meshServices {
		serviceNames = append(serviceNames, fmt.Sprintf("%s/%s", meshService.Namespace, meshService.Name))
	}
	return serviceNames
}

// listServicesForPod lists Kubernetes services whose selectors match pod labels
func listServicesForPod(pod *v1.Pod, kubeController k8s.Controller) []service.MeshService {
	var serviceList []v1.Service
	svcList := kubeController.ListServices()

	var attachedFromServiceList []*v1.Service
	attachedToServices := make(map[string]string)

	for _, svc := range svcList {
		ns := kubeController.GetNamespace(svc.Namespace)
		if ctok.IsSyncCloudNamespace(ns) {
			if len(ns.Annotations) > 0 {
				if _, exists := ns.Annotations[connector.AnnotationCloudServiceAttachedTo]; exists {
					svc := svc
					attachedFromServiceList = append(attachedFromServiceList, svc)
				}
			}
			if len(svc.Annotations) > 0 {
				if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
					svcMeta := connector.Decode(svc, v)
					if _, ok := svcMeta.Endpoints[connector.MicroEndpointAddr(pod.Status.PodIP)]; ok {
						serviceList = append(serviceList, *svc)
						attachedToServices[svc.Name] = svc.Namespace
					}
				}
			}
		} else {
			if svc.Namespace != pod.Namespace {
				continue
			}
			svcRawSelector := svc.Spec.Selector
			// service has no selectors, we do not need to match against the pod label
			if len(svcRawSelector) == 0 {
				continue
			}
			selector := labels.Set(svcRawSelector).AsSelector()
			if selector.Matches(labels.Set(pod.Labels)) {
				serviceList = append(serviceList, *svc)
			}
		}
	}

	if len(attachedToServices) > 0 && len(attachedFromServiceList) > 0 {
		for _, svc := range attachedFromServiceList {
			svcAttachedToNs, existsSvc := attachedToServices[svc.Name]
			if !existsSvc {
				continue
			}
			ns := kubeController.GetNamespace(svc.Namespace)
			if len(ns.Annotations) > 0 {
				nsAttachedToNs, existsNs := ns.Annotations[connector.AnnotationCloudServiceAttachedTo]
				if !existsNs {
					continue
				}
				if strings.EqualFold(svcAttachedToNs, nsAttachedToNs) {
					serviceList = append(serviceList, *svc)
				}
			}
		}
	}

	if len(serviceList) == 0 {
		return nil
	}

	meshServices := kubernetesServicesToMeshServices(kubeController, serviceList, pod.GetName())

	return meshServices
}

// listServicesForVm lists Kubernetes services whose selectors match vm labels
func listServicesForVm(vm *machinev1alpha1.VirtualMachine, kubeController k8s.Controller) []service.MeshService {
	var serviceList []v1.Service
	svcList := kubeController.ListServices()

	for _, svc := range svcList {
		ns := kubeController.GetNamespace(svc.Namespace)
		if ctok.IsSyncCloudNamespace(ns) {
			if len(svc.Annotations) > 0 {
				if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
					svcMeta := connector.Decode(svc, v)
					if _, ok := svcMeta.Endpoints[connector.MicroEndpointAddr(vm.Spec.MachineIP)]; ok {
						serviceList = append(serviceList, *svc)
					}
				}
			}
		} else {
			if svc.Namespace != vm.Namespace {
				continue
			}
			svcRawSelector := svc.Spec.Selector
			// service has no selectors, we do not need to match against the vm label
			if len(svcRawSelector) == 0 {
				continue
			}
			selector := labels.Set(svcRawSelector).AsSelector()
			if selector.Matches(labels.Set(vm.Labels)) {
				serviceList = append(serviceList, *svc)
			}
		}
	}

	if len(serviceList) == 0 {
		return nil
	}

	meshServices := kubernetesServicesToMeshServices(kubeController, serviceList, vm.GetName())

	return meshServices
}
