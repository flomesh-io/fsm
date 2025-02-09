package v2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("fsm-gateway/processor-v2")
)

type endpointContext struct {
	address string
	port    int32
}

type calculateBackendTargetsFunc func(svc *corev1.Service, port *int32) []fgwv2.BackendTarget

// BackendPolicyProcessor is an interface for enriching backend level policies
type BackendPolicyProcessor interface {
	Process(route client.Object, routeParentRef gwv1.ParentReference, routeRule any, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName)
}

// FilterPolicyProcessor is an interface for enriching filters for non-HTTP route rule
type FilterPolicyProcessor interface {
	Process(route client.Object, routeParentRef gwv1.ParentReference, rule *gwv1.SectionName) []gwpav1alpha2.LocalFilterReference
}

// ---

type DummySecretReferenceConditionProvider struct{}

func (r *DummySecretReferenceConditionProvider) AddInvalidCertificateRefCondition(obj client.Object, ref gwv1.SecretObjectReference) {

}

func (r *DummySecretReferenceConditionProvider) AddRefNotPermittedCondition(obj client.Object, ref gwv1.SecretObjectReference) {

}

func (r *DummySecretReferenceConditionProvider) AddRefNotFoundCondition(obj client.Object, key types.NamespacedName) {

}

func (r *DummySecretReferenceConditionProvider) AddGetRefErrorCondition(obj client.Object, key types.NamespacedName, err error) {

}

func (r *DummySecretReferenceConditionProvider) AddRefsResolvedCondition(obj client.Object) {

}

// ---

type DummyObjectReferenceConditionProvider struct{}

func (r *DummyObjectReferenceConditionProvider) AddInvalidRefCondition(obj client.Object, ref gwv1.ObjectReference) {
}

func (r *DummyObjectReferenceConditionProvider) AddRefNotPermittedCondition(obj client.Object, ref gwv1.ObjectReference) {
}

func (r *DummyObjectReferenceConditionProvider) AddRefNotFoundCondition(obj client.Object, key types.NamespacedName, kind string) {
}

func (r *DummyObjectReferenceConditionProvider) AddGetRefErrorCondition(obj client.Object, key types.NamespacedName, kind string, err error) {
}

func (r *DummyObjectReferenceConditionProvider) AddNoRequiredCAFileCondition(obj client.Object, key types.NamespacedName, kind string) {
}

func (r *DummyObjectReferenceConditionProvider) AddEmptyCACondition(obj client.Object, ref gwv1.ObjectReference) {
}

func (r *DummyObjectReferenceConditionProvider) AddRefsResolvedCondition(obj client.Object) {
}

// ---

type DummyGatewayListenerConditionProvider struct{}

func (p *DummyGatewayListenerConditionProvider) AddNoMatchingParentCondition(route client.Object, parentRef gwv1.ParentReference, routeNs string) {
}
func (p *DummyGatewayListenerConditionProvider) AddNotAllowedByListenersCondition(route client.Object, parentRef gwv1.ParentReference, routeNs string) {
}
