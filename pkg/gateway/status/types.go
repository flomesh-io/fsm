// Package status implements utility routines related to the status of the Gateway API resource.
package status

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/logger"
)

type computeParams struct {
	ParentRefs      []gwv1.ParentReference
	RouteGvk        schema.GroupVersionKind
	RouteGeneration int64
	RouteHostnames  []gwv1.Hostname
	RouteNs         string
}

var (
	log = logger.New("fsm-gateway/status")
)
