package v2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("fsm-gateway/processor-v2")
)

type serviceContext struct {
	svcPortName fgwv2.ServicePortName
}

type endpointContext struct {
	address string
	port    int32
}

type calculateBackendTargetsFunc func(svc *corev1.Service, port *int32) []fgwv2.BackendTarget

// BackendPolicyProcessor is an interface for enriching backend level policies
type BackendPolicyProcessor interface {
	Process(route client.Object, routeParentRef gwv1.ParentReference, routeRule any, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName)
}

// ---

type DummySecretReferenceResolver struct{}

func (r *DummySecretReferenceResolver) AddInvalidCertificateRefCondition(ref gwv1.SecretObjectReference) {

}

func (r *DummySecretReferenceResolver) AddRefNotPermittedCondition(ref gwv1.SecretObjectReference) {

}

func (r *DummySecretReferenceResolver) AddRefNotFoundCondition(key types.NamespacedName) {

}

func (r *DummySecretReferenceResolver) AddGetRefErrorCondition(key types.NamespacedName, err error) {

}

// ---

type DummyObjectReferenceResolver struct{}

func (r *DummyObjectReferenceResolver) AddInvalidRefCondition(ref gwv1.ObjectReference) {
}

func (r *DummyObjectReferenceResolver) AddRefNotPermittedCondition(ref gwv1.ObjectReference) {
}

func (r *DummyObjectReferenceResolver) AddRefNotFoundCondition(key types.NamespacedName, kind string) {
}

func (r *DummyObjectReferenceResolver) AddGetRefErrorCondition(key types.NamespacedName, kind string, err error) {
}

func (r *DummyObjectReferenceResolver) AddNoRequiredCAFileCondition(key types.NamespacedName, kind string) {
}

func (r *DummyObjectReferenceResolver) AddEmptyCACondition(ref gwv1.ObjectReference, refererNamespace string) {
}
