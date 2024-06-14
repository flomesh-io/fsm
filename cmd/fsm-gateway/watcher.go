package main

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type serviceStatusWatcher struct {
	serviceName string
	gateway     types.NamespacedName
	addresses   chan []gwv1.GatewayStatusAddress
	client      client.Client
	kubeClient  kubernetes.Interface
}

func (w *serviceStatusWatcher) OnAdd(obj any, _ bool) {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		// not a service
		log.Error().Msgf("Unexpected object type: %T", obj)
		return
	}

	log.Debug().Msgf("Received new service address [A]")

	w.notify(w.gatewayAddresses(svc))
}

func (w *serviceStatusWatcher) OnUpdate(_, newObj any) {
	svc, ok := newObj.(*corev1.Service)
	if !ok {
		// not a service
		log.Error().Msgf("Unexpected object type: %T", newObj)
		return
	}

	log.Debug().Msgf("Received new service address [U]")

	w.notify(w.gatewayAddresses(svc))
}

func (w *serviceStatusWatcher) OnDelete(obj any) {
	_, ok := obj.(*corev1.Service)
	if !ok {
		// not a service
		log.Error().Msgf("Unexpected object type: %T", obj)
		return
	}

	log.Debug().Msgf("Received service deletion [D]")

	w.notify(nil)
}

func (w *serviceStatusWatcher) notify(addresses []gwv1.GatewayStatusAddress) {
	w.addresses <- addresses
}

func (w *serviceStatusWatcher) shouldIgnore(obj any) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		// not a service
		log.Error().Msgf("Unexpected object type: %T", obj)
		return true
	}

	if w.serviceName == "" {
		log.Warn().Msgf("Gateway LoadBalancer Service name is empty")
		return true
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer && svc.Spec.Type != corev1.ServiceTypeNodePort {
		log.Warn().Msgf("Service %s/%s is not of type LoadBalancer or NodePort", svc.Namespace, svc.Name)
		return true
	}

	if svc.Namespace != w.gateway.Namespace {
		log.Warn().Msgf("Service namespace %q is not in the same namespace as the gateway %q", svc.Namespace, w.gateway.Namespace)
		return true
	}

	if svc.Name != w.serviceName {
		log.Warn().Msgf("Service name %q is not the same as the Gateway LoadBalancer Service name %q", svc.Name, w.serviceName)
		return true
	}

	return false
}

func (w *serviceStatusWatcher) gatewayAddresses(gwSvc *corev1.Service) []gwv1.GatewayStatusAddress {
	var addresses, hostnames []string

	switch gwSvc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		for i := range gwSvc.Status.LoadBalancer.Ingress {
			switch {
			case len(gwSvc.Status.LoadBalancer.Ingress[i].IP) > 0:
				addresses = append(addresses, gwSvc.Status.LoadBalancer.Ingress[i].IP)
			case len(gwSvc.Status.LoadBalancer.Ingress[i].Hostname) > 0:
				if gwSvc.Status.LoadBalancer.Ingress[i].Hostname == "localhost" {
					addresses = append(addresses, "127.0.0.1")
				}
				hostnames = append(hostnames, gwSvc.Status.LoadBalancer.Ingress[i].Hostname)
			}
		}
	case corev1.ServiceTypeNodePort:
		addresses = append(addresses, w.getNodeIPs(gwSvc)...)
	default:
		return nil
	}

	var gwAddresses []gwv1.GatewayStatusAddress
	for i := range addresses {
		addr := gwv1.GatewayStatusAddress{
			Type:  ptr.To(gwv1.IPAddressType),
			Value: addresses[i],
		}
		gwAddresses = append(gwAddresses, addr)
	}

	for i := range hostnames {
		addr := gwv1.GatewayStatusAddress{
			Type:  ptr.To(gwv1.HostnameAddressType),
			Value: hostnames[i],
		}
		gwAddresses = append(gwAddresses, addr)
	}

	return gwAddresses
}

func (w *serviceStatusWatcher) getNodeIPs(svc *corev1.Service) []string {
	pods := &corev1.PodList{}
	if err := w.client.List(
		context.Background(),
		pods,
		client.InNamespace(svc.Namespace),
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromSet(svc.Spec.Selector),
		},
	); err != nil {
		log.Error().Msgf("Failed to get pods: %s", err)
		return nil
	}

	extIPs := sets.New[string]()
	intIPs := sets.New[string]()

	for _, pod := range pods.Items {
		if pod.Spec.NodeName == "" || pod.Status.PodIP == "" {
			continue
		}

		if !utils.IsPodStatusConditionTrue(pod.Status.Conditions, corev1.PodReady) {
			continue
		}

		node := &corev1.Node{}
		if err := w.client.Get(context.Background(), client.ObjectKey{Name: pod.Spec.NodeName}, node); err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			log.Error().Msgf("Failed to get node %q: %s", pod.Spec.NodeName, err)
			return nil
		}

		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeExternalIP:
				extIPs.Insert(addr.Address)
			case corev1.NodeInternalIP:
				intIPs.Insert(addr.Address)
			default:
				continue
			}
		}
	}

	var nodeIPs []string
	if len(extIPs) > 0 {
		nodeIPs = extIPs.UnsortedList()
	} else {
		nodeIPs = intIPs.UnsortedList()
	}

	if version.IsDualStackEnabled(w.kubeClient) {
		ips, err := utils.FilterByIPFamily(nodeIPs, svc)
		if err != nil {
			return nil
		}

		nodeIPs = ips
	}

	return nodeIPs
}
