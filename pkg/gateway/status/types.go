// Package status implements utility routines related to the status of the Gateway API resource.
package status

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/logger"
)

type computeParams struct {
	ParentRefs      []gwv1beta1.ParentReference
	RouteGvk        schema.GroupVersionKind
	RouteGeneration int64
	RouteHostnames  []gwv1beta1.Hostname
	RouteNs         string
}

var (
	log = logger.New("fsm-gateway/status")
)
