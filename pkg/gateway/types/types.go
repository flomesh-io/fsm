// Package types contains types used by the gateway controller
package types

import (
	"github.com/flomesh-io/fsm/pkg/logger"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

var (
	log = logger.New("fsm-gateway/types")
)

// Listener is a wrapper around the Gateway API Listener object
type Listener struct {
	gwv1beta1.Listener
	SupportedKinds []gwv1beta1.RouteGroupKind
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
