package utils

import (
	"context"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/constants"
)

// GetServiceRefGrants returns all ReferenceGrants in the cache that target a Kubernetes Service
func GetServiceRefGrants(c cache.Cache) []*gwv1beta1.ReferenceGrant {
	list := &gwv1beta1.ReferenceGrantList{}
	err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.KubernetesServiceKind),
	})
	if err != nil {
		log.Error().Msgf("Failed to list ReferenceGrants: %v", err)
		return nil
	}

	return ToSlicePtr(list.Items)
}

// GetSecretRefGrants returns all ReferenceGrants in the cache that target a Kubernetes Secret
func GetSecretRefGrants(c cache.Cache) []*gwv1beta1.ReferenceGrant {
	list := &gwv1beta1.ReferenceGrantList{}
	err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.KubernetesSecretKind),
	})
	if err != nil {
		log.Error().Msgf("Failed to list ReferenceGrants: %v", err)
		return nil
	}

	log.Warn().Msgf("SecretRefGrants: %#v", list.Items)

	return ToSlicePtr(list.Items)
}

// GetCARefGrants returns all ReferenceGrants in the cache that target a Kubernetes Secret or ConfigMap
func GetCARefGrants(c cache.Cache) []*gwv1beta1.ReferenceGrant {
	list := &gwv1beta1.ReferenceGrantList{}
	err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: gwtypes.OrSelectors(
			fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.KubernetesSecretKind),
			fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.KubernetesConfigMapKind),
		),
	})
	if err != nil {
		log.Error().Msgf("Failed to list ReferenceGrants: %v", err)
		return nil
	}

	return ToSlicePtr(list.Items)
}
