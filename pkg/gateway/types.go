// Package gateway implements the Kubernetes client for the resources in the gateway.networking.k8s.io API group
package gateway

import (
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"time"

	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// client is the type used to represent the Kubernetes client for the gateway.networking.k8s.io API group
type client struct {
	informers  *informers.InformerCollection
	kubeClient kubernetes.Interface
	msgBroker  *messaging.Broker
	cache      Cache
	cfg        configurator.Configurator
}

// Controller is the interface for the functionality provided by the resources part of the gateway.networking.k8s.io API group
type Controller interface {
	cache.ResourceEventHandler

	// Start runs the backend broadcast listener
	Start() error
}

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	// This is set to 0 because we do not need resyncs from k8s client, and have our
	// own Ticker to turn on periodic resyncs.
	DefaultKubeEventResyncInterval = 0 * time.Second
)

type Cache interface {
	Insert(obj interface{}) bool
	Delete(obj interface{}) bool
	BuildConfigs()
}

type Listener struct {
	gwv1beta1.Listener
	SupportedKinds []gwv1beta1.RouteGroupKind
}

func (l *Listener) AllowsKind(gvk schema.GroupVersionKind) bool {
	for _, allowedKind := range l.SupportedKinds {
		kind := gwv1beta1.Kind(gvk.Kind)
		group := gwv1beta1.Group(gvk.Group)

		if allowedKind.Kind == kind && *allowedKind.Group == group {
			return true
		}
	}

	return false
}

type ComputeParams struct {
	ParentRefs      []gwv1beta1.ParentReference
	RouteGvk        schema.GroupVersionKind
	RouteGeneration int64
	RouteHostnames  []gwv1beta1.Hostname
}
