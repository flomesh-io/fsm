package machine

import (
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/announcements"
	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// NewMachineController returns a machine.Controller interface related to functionality provided by the resources in the machine.flomesh.io API group
func NewMachineController(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, kubeController k8s.Controller, msgBroker *messaging.Broker) *Client {
	client := &Client{
		informers:      informerCollection,
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	shouldObserveVM := func(obj interface{}) bool {
		return true
	}

	vmEventTypes := k8s.EventTypes{
		Add:    announcements.VirtualMachineAdded,
		Update: announcements.VirtualMachineUpdated,
		Delete: announcements.VirtualMachineDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyVirtualMachine, k8s.GetEventHandlerFuncs(shouldObserveVM, vmEventTypes, msgBroker))

	return client
}

// GetVms lists vms
func (c *Client) GetVms() []*machinev1alpha1.VirtualMachine {
	var vms []*machinev1alpha1.VirtualMachine
	for _, vmIface := range c.informers.List(informers.InformerKeyVirtualMachine) {
		vm := vmIface.(*machinev1alpha1.VirtualMachine)
		vms = append(vms, vm)
	}
	return vms
}
