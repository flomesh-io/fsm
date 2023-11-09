// +k8s:deepcopy-gen=package,register
// +groupName=gateway.flomesh.io

// Package v1alpha1 contains API Schema definitions for the gateway.flomesh.io v1alpha1 API group
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/flomesh-io/fsm/pkg/constants"
)

var (
	// SchemeGroupVersion is group version used to register MeshConfig
	SchemeGroupVersion = schema.GroupVersion{
		Group:   constants.FlomeshGatewayAPIGroup,
		Version: "v1alpha1",
	}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds all Resources to the Scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&RateLimitPolicy{},
		&RateLimitPolicyList{},
		&SessionStickyPolicy{},
		&SessionStickyPolicyList{},
		&LoadBalancerPolicy{},
		&LoadBalancerPolicyList{},
		&CircuitBreakingPolicy{},
		&CircuitBreakingPolicyList{},
		&AccessControlPolicy{},
		&AccessControlPolicyList{},
		&HealthCheckPolicy{},
		&HealthCheckPolicyList{},
		&FaultInjectionPolicy{},
		&FaultInjectionPolicyList{},
		&UpstreamTLSPolicy{},
		&UpstreamTLSPolicyList{},
		&RetryPolicy{},
		&RetryPolicyList{},
		&GatewayTLSPolicy{},
		&GatewayTLSPolicyList{},
	)

	metav1.AddToGroupVersion(
		scheme,
		SchemeGroupVersion,
	)
	return nil
}
