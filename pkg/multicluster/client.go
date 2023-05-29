package multicluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	multiclusterv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	networkingv1 "github.com/flomesh-io/fsm/pkg/apis/networking/v1"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// NewMultiClusterController returns a multicluster.Controller interface related to functionality provided by the resources in the flomesh.io API group
func NewMultiClusterController(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, kubeController k8s.Controller, msgBroker *messaging.Broker) *Client {
	client := &Client{
		informers:      informerCollection,
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		if _, object := obj.(metav1.Object); !object {
			return false
		}
		if _, serviceExport := obj.(*multiclusterv1alpha1.ServiceExport); serviceExport {
			return true
		}
		if _, serviceImport := obj.(*multiclusterv1alpha1.ServiceImport); serviceImport {
			return true
		}
		if _, gblTrafficPolicy := obj.(*multiclusterv1alpha1.GlobalTrafficPolicy); gblTrafficPolicy {
			return true
		}
		_, ingressClass := obj.(*networkingv1.IngressClass)
		return ingressClass
	}

	svcExportEventTypes := k8s.EventTypes{
		Add:    announcements.ServiceExportAdded,
		Update: announcements.ServiceExportUpdated,
		Delete: announcements.ServiceExportDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyServiceExport, k8s.GetEventHandlerFuncs(shouldObserve, svcExportEventTypes, msgBroker))

	svcImportEventTypes := k8s.EventTypes{
		Add:    announcements.ServiceImportAdded,
		Update: announcements.ServiceImportUpdated,
		Delete: announcements.ServiceImportDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyServiceImport, k8s.GetEventHandlerFuncs(shouldObserve, svcImportEventTypes, msgBroker))

	glbTrafficPolicyTypes := k8s.EventTypes{
		Add:    announcements.GlobalTrafficPolicyAdded,
		Update: announcements.GlobalTrafficPolicyUpdated,
		Delete: announcements.GlobalTrafficPolicyDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyGlobalTrafficPolicy, k8s.GetEventHandlerFuncs(shouldObserve, glbTrafficPolicyTypes, msgBroker))

	ingressClassEventTypes := k8s.EventTypes{
		Add:    announcements.IngressClassAdded,
		Update: announcements.IngressClassUpdated,
		Delete: announcements.IngressClassDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyIngressClass, k8s.GetEventHandlerFuncs(shouldObserve, ingressClassEventTypes, msgBroker))

	return client
}
