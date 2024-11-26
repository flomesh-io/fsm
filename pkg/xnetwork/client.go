package xnetwork

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	xnetv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/xnetwork/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// NewXNetworkController returns a xnetwork.Controller interface related to functionality provided by the resources in the xnetwork.flomesh.io API group
func NewXNetworkController(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, kubeController k8s.Controller, msgBroker *messaging.Broker) Controller {
	client := &Client{
		informers:      informerCollection,
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return kubeController.IsMonitoredNamespace(object.GetNamespace())
	}

	xAccessControlEventTypes := k8s.EventTypes{
		Add:    announcements.XAccessControlAdded,
		Update: announcements.XAccessControlUpdated,
		Delete: announcements.XAccessControlDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyXNetworkAccessControl,
		k8s.GetEventHandlerFuncs(shouldObserve, xAccessControlEventTypes, msgBroker))

	svcEventTypes := k8s.EventTypes{
		Add:    announcements.ServiceAdded,
		Update: announcements.ServiceUpdated,
		Delete: announcements.ServiceDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyService,
		k8s.GetEventHandlerFuncs(func(obj interface{}) bool {
			service, ok := obj.(corev1.Service)
			if !ok {
				return false
			}
			return client.isAccessControlService(service.Namespace, service.Name)
		}, svcEventTypes, msgBroker))

	eptEventTypes := k8s.EventTypes{
		Add:    announcements.EndpointAdded,
		Update: announcements.EndpointUpdated,
		Delete: announcements.EndpointDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyEndpoints,
		k8s.GetEventHandlerFuncs(func(obj interface{}) bool {
			eps, ok := obj.(corev1.Endpoints)
			if !ok {
				return false
			}
			return client.isAccessControlService(eps.Namespace, eps.Name)
		}, eptEventTypes, msgBroker))

	return client
}

func (c *Client) isAccessControlService(namespace, name string) bool {
	for _, accessControlIface := range c.informers.List(informers.InformerKeyXNetworkAccessControl) {
		accessControl := accessControlIface.(*xnetv1alpha1.AccessControl)
		if len(accessControl.Spec.Services) > 0 {
			for _, aclService := range accessControl.Spec.Services {
				if strings.EqualFold(aclService.Name, name) {
					if len(aclService.Namespace) > 0 {
						if strings.EqualFold(aclService.Namespace, namespace) {
							return true
						}
					} else {
						if strings.EqualFold(accessControl.Namespace, namespace) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// GetAccessControls lists AccessControls
func (c *Client) GetAccessControls() []*xnetv1alpha1.AccessControl {
	var accessControls []*xnetv1alpha1.AccessControl
	for _, accessControlIface := range c.informers.List(informers.InformerKeyXNetworkAccessControl) {
		accessControl := accessControlIface.(*xnetv1alpha1.AccessControl)
		accessControls = append(accessControls, accessControl)
	}
	return accessControls
}
