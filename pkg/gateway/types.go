// Package gateway implements the Kubernetes client for the resources in the gateway.networking.k8s.io API group
package gateway

import (
	"github.com/flomesh-io/fsm/pkg/messaging"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
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
}

// Controller is the interface for the functionality provided by the resources part of the gateway.networking.k8s.io API group
type Controller interface {
	// Start runs the backend broadcast listener
	Start() error

	//// GetEffectiveGatewayClass returns the active and accepted gatewayclasses
	//GetEffectiveGatewayClass() *gwv1beta1.GatewayClass
	//
	//// GetEffectiveGateways lists effective gateways attached to effective GatewayClass
	//GetEffectiveGateways() []*gwv1beta1.Gateway
	//
	//// GetHTTPRoutes lists httproutes
	//GetHTTPRoutes() []*gwv1beta1.HTTPRoute
	//
	//// GetGRPCRoutes lists grpcroutes
	//GetGRPCRoutes() []*gwv1alpha2.GRPCRoute
	//
	//// GetTLSRoutes lists tlsroutes
	//GetTLSRoutes() []*gwv1alpha2.TLSRoute
	//
	//// GetTCPRoutes lists tcproutes
	//GetTCPRoutes() []*gwv1alpha2.TCPRoute
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
