// Package types contains types used by the gateway controller
package types

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// Listener is a wrapper around the Gateway API Listener object
type Listener struct {
	gwv1beta1.Listener
	SupportedKinds []gwv1beta1.RouteGroupKind
}

// AllowsKind returns true if the listener allows the given kind
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
