package registry

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/ctok"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
)

// ProxyServiceMapper knows how to map Sidecar instances to services.
type ProxyServiceMapper interface {
	ListProxyServices(*pipy.Proxy) ([]service.MeshService, error)
}

// ExplicitProxyServiceMapper is a custom ProxyServiceMapper implementation.
type ExplicitProxyServiceMapper func(*pipy.Proxy) ([]service.MeshService, error)

// ListProxyServices executes the given mapping.
func (e ExplicitProxyServiceMapper) ListProxyServices(p *pipy.Proxy) ([]service.MeshService, error) {
	return e(p)
}

// KubeProxyServiceMapper maps an Sidecar instance to services in a Kubernetes cluster.
type KubeProxyServiceMapper struct {
	KubeController k8s.Controller
	Configurator   configurator.Configurator
}

// ListProxyServices maps an Pipy instance to a number of Kubernetes services.
func (k *KubeProxyServiceMapper) ListProxyServices(p *pipy.Proxy) ([]service.MeshService, error) {
	var meshServices []service.MeshService
	if p.VM {
		vm, err := k.KubeController.GetVmForProxy(p)
		if err != nil {
			return nil, err
		}

		p.MachineIP = pipy.NewNetAddress(vm.Spec.MachineIP)
		if _, internal := vm.Annotations[connector.AnnotationMeshServiceInternalSync]; internal {
			p.ClusterID = ""
		} else {
			if clusterId, exists := vm.Annotations[connector.AnnotationCloudServiceInheritedClusterID]; exists {
				p.ClusterID = clusterId
			}
		}

		meshServices = listServicesForVm(vm, k.KubeController, k.Configurator)

		servicesForPod := strings.Join(listServiceNames(meshServices), ",")
		log.Trace().Msgf("Services associated with VM with UID=%s Name=%s/%s: %+v",
			vm.UID, vm.Namespace, vm.Name, servicesForPod)
	} else {
		pod, err := k.KubeController.GetPodForProxy(p)
		if err != nil {
			return nil, err
		}

		meshServices = listServicesForPod(pod, k.KubeController, k.Configurator)

		servicesForPod := strings.Join(listServiceNames(meshServices), ",")
		log.Trace().Msgf("Services associated with Pod with UID=%s Name=%s/%s: %+v",
			pod.UID, pod.Namespace, pod.Name, servicesForPod)
	}

	return meshServices, nil
}

func kubernetesServicesToMeshServices(kubeController k8s.Controller, cfg configurator.Configurator, kubernetesServices []corev1.Service, subdomainFilter string) (meshServices []service.MeshService) {
	for _, svc := range kubernetesServices {
		svc := svc
		for _, meshSvc := range k8s.ServiceToMeshServices(kubeController, cfg.GetMeshConfig().Spec.Connector.Lb, &svc) {
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
func listServicesForPod(pod *corev1.Pod, kubeController k8s.Controller, cfg configurator.Configurator) []service.MeshService {
	var serviceList []corev1.Service
	svcList := kubeController.ListServices(true, true)

	var attachedFromServiceList []*corev1.Service
	attachedToServices := make(map[string]string)

	lb := cfg.GetMeshConfig().Spec.Connector.Lb

	for _, svc := range svcList {
		ns := kubeController.GetNamespace(svc.Namespace)
		if ctok.IsSyncCloudNamespace(ns) {
			isSlaveNamespace := lb.IsSlaveNamespace(ns.Name)
			if !isSlaveNamespace {
				if len(ns.Annotations) > 0 {
					_, isSlaveNamespace = ns.Annotations[connector.AnnotationCloudServiceAttachedTo]
				}
			}
			if isSlaveNamespace {
				svc := svc
				attachedFromServiceList = append(attachedFromServiceList, svc)
			}

			if len(svc.Annotations) > 0 {
				if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
					svcMeta := connector.Decode(svc, v)
					if _, ok := svcMeta.Endpoints[connector.MicroServiceAddress(pod.Status.PodIP)]; ok {
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
			if lb.IsSlaveNamespace(svc.Namespace) && lb.IsMasterNamespace(svcAttachedToNs) {
				serviceList = append(serviceList, *svc)
			} else {
				ns := kubeController.GetNamespace(svc.Namespace)
				if len(ns.Annotations) > 0 {
					if nsAttachedToNs, existsNs := ns.Annotations[connector.AnnotationCloudServiceAttachedTo]; existsNs {
						if strings.EqualFold(svcAttachedToNs, nsAttachedToNs) {
							serviceList = append(serviceList, *svc)
						}
					}
				}
			}
		}
	}

	if len(serviceList) == 0 {
		return nil
	}

	meshServices := kubernetesServicesToMeshServices(kubeController, cfg, serviceList, pod.GetName())

	return meshServices
}

// listServicesForVm lists Kubernetes services whose selectors match vm labels
func listServicesForVm(vm *machinev1alpha1.VirtualMachine, kubeController k8s.Controller, cfg configurator.Configurator) []service.MeshService {
	var serviceList []corev1.Service
	svcList := kubeController.ListServices(true, true)

	for _, svc := range svcList {
		ns := kubeController.GetNamespace(svc.Namespace)
		if ctok.IsSyncCloudNamespace(ns) {
			if len(svc.Annotations) > 0 {
				if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
					svcMeta := connector.Decode(svc, v)
					if _, ok := svcMeta.Endpoints[connector.MicroServiceAddress(vm.Spec.MachineIP)]; ok {
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

	meshServices := kubernetesServicesToMeshServices(kubeController, cfg, serviceList, vm.GetName())

	return meshServices
}
