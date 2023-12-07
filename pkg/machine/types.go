// Package machine implements the Kubernetes client for the resources in the machine.flomesh.io API group
package machine

import (
	"k8s.io/client-go/kubernetes"

	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// Client is the type used to represent the Kubernetes Client for the machine.flomesh.io API group
type Client struct {
	informers      *informers.InformerCollection
	kubeClient     kubernetes.Interface
	kubeController k8s.Controller
}

// Controller is the interface for the functionality provided by the resources part of the machine.flomesh.io API group
type Controller interface {
	// GetVms lists vms
	GetVms() []*machinev1alpha1.VirtualMachine
}
