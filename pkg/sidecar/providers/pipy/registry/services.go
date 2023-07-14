package registry

import (
	"fmt"
	"net"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// ProxyServiceMapper knows how to map Sidecar instances to services.
type ProxyServiceMapper interface {
	ListProxyServices(*pipy.Proxy) ([]service.MeshService, error)
	ListProxyPods() []v1.Pod
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
}

// ListProxyPods maps an Sidecar instance to a number of Kubernetes services.
func (k *KubeProxyServiceMapper) ListProxyPods() []v1.Pod {
	allPods := k.KubeController.ListPods()
	var matchedPods []v1.Pod
	for _, pod := range allPods {
		if _, exists := pod.Labels[constants.SidecarUniqueIDLabelName]; exists {
			matchedPods = append(matchedPods, *pod)
		}
	}
	return matchedPods
}

// ListProxyServices maps an Pipy instance to a number of Kubernetes services.
func (k *KubeProxyServiceMapper) ListProxyServices(p *pipy.Proxy) ([]service.MeshService, error) {
	pod, err := k.KubeController.GetPodForProxy(p)
	if err != nil {
		return nil, err
	}

	meshServices := listServicesForPod(pod, k.KubeController)

	servicesForPod := strings.Join(listServiceNames(meshServices), ",")
	log.Trace().Msgf("Services associated with Pod with UID=%s Name=%s/%s: %+v",
		pod.ObjectMeta.UID, pod.Namespace, pod.Name, servicesForPod)

	return meshServices, nil
}

func kubernetesServicesToMeshServices(kubeController k8s.Controller, kubernetesServices []v1.Service, subdomainFilter string) (meshServices []service.MeshService) {
	for _, svc := range kubernetesServices {
		for _, meshSvc := range k8s.ServiceToMeshServices(kubeController, svc) {
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

	for _, svc := range svcList {
		if connector.IsSyncCloudNamespace(svc.Namespace) {
			if len(svc.Annotations) > 0 {
				if _, exists := svc.ObjectMeta.Annotations[fmt.Sprintf("%s-%d", connector.MeshEndpointAddrAnnotation, utils.IP2Int(net.ParseIP(pod.Status.PodIP).To4()))]; exists {
					serviceList = append(serviceList, *svc)
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

	if len(serviceList) == 0 {
		return nil
	}

	meshServices := kubernetesServicesToMeshServices(kubeController, serviceList, pod.GetName())

	return meshServices
}
