// Package types contains types used by the gateway controller
package types

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime/schema"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("fsm-gateway/types")
)

// Controller is the interface for the functionality provided by the resources part of the gateway.networking.k8s.io API group
type Controller interface {
	cache.ResourceEventHandler

	// Runnable runs the backend broadcast listener
	manager.Runnable

	// LeaderElectionRunnable knows if a Runnable needs to be run in the leader election mode.
	manager.LeaderElectionRunnable
}

// PolicyMatchType is the type used to represent the rate limit policy match type
type PolicyMatchType string

const (
	// PolicyMatchTypePort is the type used to represent the rate limit policy match type port
	PolicyMatchTypePort PolicyMatchType = "port"

	// PolicyMatchTypeHostnames is the type used to represent the rate limit policy match type hostnames
	PolicyMatchTypeHostnames PolicyMatchType = "hostnames"

	// PolicyMatchTypeHTTPRoute is the type used to represent the rate limit policy match type httproute
	PolicyMatchTypeHTTPRoute PolicyMatchType = "httproute"

	// PolicyMatchTypeGRPCRoute is the type used to represent the rate limit policy match type grpcroute
	PolicyMatchTypeGRPCRoute PolicyMatchType = "grpcroute"
)

// Listener is a wrapper around the Gateway API Listener object
type Listener struct {
	gwv1.Listener
	SupportedKinds []gwv1.RouteGroupKind
}

// AllowsKind returns true if the listener allows the given kind
func (l *Listener) AllowsKind(gvk schema.GroupVersionKind) bool {
	log.Debug().Msgf("[GW-CACHE] Checking if listener allows kind %s", gvk.String())
	kind := gvk.Kind
	group := gvk.Group

	for _, allowedKind := range l.SupportedKinds {
		log.Debug().Msgf("[GW-CACHE] allowedKind={%s, %s}", *allowedKind.Group, allowedKind.Kind)
		if string(allowedKind.Kind) == kind &&
			(allowedKind.Group == nil || string(*allowedKind.Group) == group) {
			return true
		}
	}

	return false
}

// CrossNamespaceFrom is the type used to represent the from part of a cross-namespace reference
type CrossNamespaceFrom struct {
	Group     string
	Kind      string
	Namespace string
}

// CrossNamespaceTo is the type used to represent the to part of a cross-namespace reference
type CrossNamespaceTo struct {
	Group     string
	Kind      string
	Namespace string
	Name      string
}

// SecretReferenceConditionProvider is the interface for providing SecretReference conditions
type SecretReferenceConditionProvider interface {
	AddInvalidCertificateRefCondition(obj client.Object, ref gwv1.SecretObjectReference)
	AddRefNotPermittedCondition(obj client.Object, ref gwv1.SecretObjectReference)
	AddRefNotFoundCondition(obj client.Object, key types.NamespacedName)
	AddGetRefErrorCondition(obj client.Object, key types.NamespacedName, err error)
	AddRefsResolvedCondition(obj runtime.Object)
}

// ObjectReferenceConditionProvider is the interface for providing ObjectReference conditions
type ObjectReferenceConditionProvider interface {
	AddInvalidRefCondition(obj client.Object, ref gwv1.ObjectReference)
	AddRefNotPermittedCondition(obj client.Object, ref gwv1.ObjectReference)
	AddRefNotFoundCondition(obj client.Object, key types.NamespacedName, kind string)
	AddGetRefErrorCondition(obj client.Object, key types.NamespacedName, kind string, err error)
	AddNoRequiredCAFileCondition(obj client.Object, key types.NamespacedName, kind string)
	AddEmptyCACondition(obj client.Object, ref gwv1.ObjectReference)
	AddRefsResolvedCondition(obj runtime.Object)
}

// SecretReferenceResolver is the interface for resolving SecretReferences
type SecretReferenceResolver interface {
	ResolveAllRefs(referer client.Object, refs []gwv1.SecretObjectReference) bool
	SecretRefToSecret(referer client.Object, ref gwv1.SecretObjectReference) (*corev1.Secret, error)
}

// ObjectReferenceResolver is the interface for resolving ObjectReferences
type ObjectReferenceResolver interface {
	ResolveAllRefs(referer client.Object, refs []gwv1.ObjectReference) bool
	ObjectRefToCACertificate(referer client.Object, ref gwv1.ObjectReference) []byte
}

// GatewayListenerConditionProvider is the interface for providing GatewayListener conditions
type GatewayListenerConditionProvider interface {
	AddNoMatchingParentCondition(route client.Object, parentRef gwv1.ParentReference, routeNs string)
	AddNotAllowedByListeners(route client.Object, parentRef gwv1.ParentReference, routeNs string)
}

// GatewayListenerResolver is the interface for resolving Listeners of the Gateway
type GatewayListenerResolver interface {
	GetAllowedListeners(gw *gwv1.Gateway) []Listener
}
