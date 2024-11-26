// Package xnetwork implements the Kubernetes client for the resources in the xnetwork.flomesh.io API group
package xnetwork

import (
	"k8s.io/client-go/kubernetes"

	xnetv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/xnetwork/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// Client is the type used to represent the Kubernetes Client for the xnetwork.flomesh.io API group
type Client struct {
	informers      *informers.InformerCollection
	kubeClient     kubernetes.Interface
	kubeController k8s.Controller
}

// Controller is the interface for the functionality provided by the resources part of the xnetwork.flomesh.io API group
type Controller interface {
	// GetAccessControls lists AccessControls
	GetAccessControls() []*xnetv1alpha1.AccessControl
}
