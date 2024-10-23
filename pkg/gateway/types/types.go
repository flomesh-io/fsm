// Package types contains types used by the gateway controller
package types

import (
	"k8s.io/apimachinery/pkg/types"
	cache "k8s.io/client-go/tools/cache"
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

// SecretReferenceResolver is the interface for resolving Secret references
type SecretReferenceResolver interface {
	AddInvalidCertificateRefCondition(ref gwv1.SecretObjectReference)
	AddRefNotPermittedCondition(ref gwv1.SecretObjectReference)
	AddRefNotFoundCondition(key types.NamespacedName)
	AddGetRefErrorCondition(key types.NamespacedName, err error)
	AddRefsResolvedCondition()
}

// ObjectReferenceResolver is the interface for resolving Object references
type ObjectReferenceResolver interface {
	AddInvalidRefCondition(ref gwv1.ObjectReference)
	AddRefNotPermittedCondition(ref gwv1.ObjectReference)
	AddRefNotFoundCondition(key types.NamespacedName, kind string)
	AddGetRefErrorCondition(key types.NamespacedName, kind string, err error)
	AddNoRequiredCAFileCondition(key types.NamespacedName, kind string)
	AddEmptyCACondition(ref gwv1.ObjectReference, refererNamespace string)
	AddRefsResolvedCondition()
}
