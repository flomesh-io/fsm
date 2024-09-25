package utils

import (
	"context"
	"fmt"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"github.com/google/go-cmp/cmp"

	"k8s.io/apimachinery/pkg/fields"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

// BackendRefToServicePortName converts a BackendRef to a ServicePortName for a given Route if the referent is a Kubernetes Service and the port is valid.
func BackendRefToServicePortName(client cache.Cache, route client.Object, backendRef gwv1.BackendObjectReference, rps status.RouteConditionAccessor) *fgwv2.ServicePortName {
	if !IsValidBackendRefToGroupKindOfService(backendRef) {
		log.Error().Msgf("Unsupported backend group %s and kind %s for service", *backendRef.Group, *backendRef.Kind)
		rps.AddCondition(
			gwv1.RouteConditionResolvedRefs,
			metav1.ConditionFalse,
			gwv1.RouteReasonInvalidKind,
			fmt.Sprintf("Unsupported backend group %s and kind %s for service", *backendRef.Group, *backendRef.Kind),
		)

		return nil
	}

	// should not happen, there's validation rules in the CRD and webhooks, just double check
	if backendRef.Port == nil {
		log.Warn().Msgf("Port is not specified in the backend reference %s/%s when the referent is a Kubernetes Service", NamespaceDerefOr(backendRef.Namespace, route.GetNamespace()), backendRef.Name)
		rps.AddCondition(
			gwv1.RouteConditionResolvedRefs,
			metav1.ConditionFalse,
			gwv1.RouteReasonBackendNotFound,
			fmt.Sprintf("Port is not specified in the backend reference %s/%s when the referent is a Kubernetes Service", NamespaceDerefOr(backendRef.Namespace, route.GetNamespace()), backendRef.Name),
		)
		return nil
	}

	gvk := route.GetObjectKind().GroupVersionKind()
	routeNamespace := route.GetNamespace()
	if backendRef.Namespace != nil && string(*backendRef.Namespace) != routeNamespace && !ValidCrossNamespaceRef(
		gwtypes.CrossNamespaceFrom{
			Group:     gvk.Group,
			Kind:      gvk.Kind,
			Namespace: routeNamespace,
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(*backendRef.Group),
			Kind:      string(*backendRef.Kind),
			Namespace: string(*backendRef.Namespace),
			Name:      string(backendRef.Name),
		},
		GetServiceRefGrants(client),
	) {
		//log.Error().Msgf("Cross-namespace reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
		//	gvk.Kind, gvk.Group, routeNamespace, route.GetName(),
		//	string(*backendRef.Kind), string(*backendRef.Group), string(*backendRef.Namespace), backendRef.Name)

		rps.AddCondition(
			gwv1.RouteConditionResolvedRefs,
			metav1.ConditionFalse,
			gwv1.RouteReasonRefNotPermitted,
			fmt.Sprintf("Backend reference to %s/%s is not allowed", string(*backendRef.Namespace), backendRef.Name),
		)

		return nil
	}

	key := types.NamespacedName{
		Namespace: NamespaceDerefOr(backendRef.Namespace, routeNamespace),
		Name:      string(backendRef.Name),
	}

	getServiceFromCache := func(key types.NamespacedName) (*corev1.Service, error) {
		obj := &corev1.Service{}
		if err := client.Get(context.Background(), key, obj); err != nil {
			return nil, err
		}

		return obj, nil
	}

	service, err := getServiceFromCache(key)
	if err != nil {
		log.Error().Msgf("Failed to get service %s: %s", key, err)

		if errors.IsNotFound(err) {
			rps.AddCondition(
				gwv1.RouteConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.RouteReasonBackendNotFound,
				fmt.Sprintf("Backend ref to Service %s not found", key),
			)
		} else {
			rps.AddCondition(
				gwv1.RouteConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.RouteReasonBackendNotFound,
				fmt.Sprintf("Failed to get service %s: %s", key, err),
			)
		}

		return nil
	}

	servicePort := func(service *corev1.Service, backendRef gwv1.BackendObjectReference) *corev1.ServicePort {
		for i, svcPort := range service.Spec.Ports {
			if svcPort.Port == int32(*backendRef.Port) {
				return &service.Spec.Ports[i]
			}
		}

		return nil
	}

	svcPort := servicePort(service, backendRef)

	if svcPort == nil {
		log.Error().Msgf("Port %d is not found in service %s", *backendRef.Port, key)
		rps.AddCondition(
			gwv1.RouteConditionResolvedRefs,
			metav1.ConditionFalse,
			gwv1.RouteReasonBackendNotFound,
			fmt.Sprintf("Port %d is not found in service %s", *backendRef.Port, key),
		)

		return nil
	}

	rps.AddCondition(
		gwv1.RouteConditionResolvedRefs,
		metav1.ConditionTrue,
		gwv1.RouteReasonResolvedRefs,
		fmt.Sprintf("References of %s is resolved", gvk.Kind),
	)

	return &fgwv2.ServicePortName{
		NamespacedName: key,
		SectionName:    svcPort.Name,
		Port:           ptr.To(int32(*backendRef.Port)),
	}
}

// FindBackendTLSPolicy finds the BackendTLSPolicy for the given LocalPolicyTargetReferenceWithSectionName.
func FindBackendTLSPolicy(c cache.Cache, targetRef gwv1alpha2.LocalPolicyTargetReferenceWithSectionName, routeNamespace string) (*gwv1alpha3.BackendTLSPolicy, bool) {
	var fallbackBackendTLSPolicy *gwv1alpha3.BackendTLSPolicy

	list := &gwv1alpha3.BackendTLSPolicyList{}
	if err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, types.NamespacedName{
			Namespace: routeNamespace,
			Name:      string(targetRef.Name),
		}.String()),
		Namespace: routeNamespace,
	}); err != nil {
		return nil, false
	}

	for _, policy := range ToSlicePtr(list.Items) {
		for _, ref := range policy.Spec.TargetRefs {
			sectionNameMatches := ref.SectionName != nil && targetRef.SectionName != nil &&
				*ref.SectionName == *targetRef.SectionName

			// Compare the LocalPolicyTargetReference
			if ref.LocalPolicyTargetReference.Group == targetRef.Group &&
				ref.LocalPolicyTargetReference.Kind == targetRef.Kind &&
				ref.LocalPolicyTargetReference.Name == targetRef.Name {
				if sectionNameMatches {
					return policy, true
				}

				if ref.SectionName == nil {
					fallbackBackendTLSPolicy = policy
				}
			}
		}
	}

	if fallbackBackendTLSPolicy != nil {
		return fallbackBackendTLSPolicy, true
	}

	return nil, false
}

// FindBackendLBPolicy finds the BackendTLSPolicy for the given LocalPolicyTargetReference.
func FindBackendLBPolicy(c cache.Cache, targetRef gwv1alpha2.LocalPolicyTargetReference, routeNamespace string) (*gwpav1alpha2.BackendLBPolicy, bool) {
	list := &gwpav1alpha2.BackendLBPolicyList{}
	if err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, types.NamespacedName{
			Namespace: routeNamespace,
			Name:      string(targetRef.Name),
		}.String()),
		Namespace: routeNamespace,
	}); err != nil {
		return nil, false
	}

	for _, policy := range ToSlicePtr(list.Items) {
		for _, ref := range policy.Spec.TargetRefs {
			// Compare the LocalPolicyTargetReference
			if cmp.Equal(ref, targetRef) {
				return policy, true
			}
		}
	}

	return nil, false
}

func FindHealthCheckPolicy(c cache.Cache, targetRef gwv1alpha2.NamespacedPolicyTargetReference, routeNamespace string, svcPort *fgwv2.ServicePortName) (*gwpav1alpha2.HealthCheckPolicy, *gwpav1alpha2.PortHealthCheck, bool) {
	if svcPort == nil {
		return nil, nil, false
	}

	if svcPort.Port == nil {
		return nil, nil, false
	}

	list := &gwpav1alpha2.HealthCheckPolicyList{}
	if err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, types.NamespacedName{
			Namespace: NamespaceDerefOr(targetRef.Namespace, routeNamespace),
			Name:      string(targetRef.Name),
		}.String()),
	}); err != nil {
		return nil, nil, false
	}

	for _, policy := range ToSlicePtr(list.Items) {
		for _, ref := range policy.Spec.TargetRefs {
			if cmp.Equal(ref, targetRef) {
				for i, port := range policy.Spec.Ports {
					if *svcPort.Port == int32(port.Port) {
						return policy, &policy.Spec.Ports[i], true
					}
				}
			}
		}
	}

	return nil, nil, false
}

func FindRetryPolicy(c cache.Cache, targetRef gwv1alpha2.NamespacedPolicyTargetReference, routeNamespace string, svcPort *fgwv2.ServicePortName) (*gwpav1alpha2.RetryPolicy, *gwpav1alpha2.PortRetry, bool) {
	if svcPort == nil {
		return nil, nil, false
	}

	if svcPort.Port == nil {
		return nil, nil, false
	}

	list := &gwpav1alpha2.RetryPolicyList{}
	if err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, types.NamespacedName{
			Namespace: NamespaceDerefOr(targetRef.Namespace, routeNamespace),
			Name:      string(targetRef.Name),
		}.String()),
	}); err != nil {
		return nil, nil, false
	}

	for _, policy := range ToSlicePtr(list.Items) {
		for _, ref := range policy.Spec.TargetRefs {
			if cmp.Equal(ref, targetRef) {
				for i, port := range policy.Spec.Ports {
					if *svcPort.Port == int32(port.Port) {
						return policy, &policy.Spec.Ports[i], true
					}
				}
			}
		}
	}

	return nil, nil, false
}
