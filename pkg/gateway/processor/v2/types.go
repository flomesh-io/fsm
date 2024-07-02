package v2

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("fsm-gateway/processor-v2")
)

type serviceContext struct {
	svcPortName v2.ServicePortName
}

type endpointContext struct {
	address string
	port    int32
}

type calculateBackendTargetsFunc func(svc *corev1.Service, port *int32) []v2.BackendTarget

// BackendPolicyProcessor is an interface for enriching backend level policies
type BackendPolicyProcessor interface {
	Process(route client.Object, routeParentRef gwv1.ParentReference, backendRef gwv1.BackendObjectReference, svcPort *v2.ServicePortName)
}
