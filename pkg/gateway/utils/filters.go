package utils

import (
	"context"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
)

// ExtensionRefToFilter converts a LocalObjectReference to a Filter.
func ExtensionRefToFilter(client cache.Cache, route client.Object, extensionRef *gwv1.LocalObjectReference) *extv1alpha1.Filter {
	key := types.NamespacedName{
		Namespace: route.GetNamespace(),
		Name:      string(extensionRef.Name),
	}
	filter := &extv1alpha1.Filter{}
	if err := client.Get(context.Background(), key, filter); err != nil {
		log.Error().Err(err).Msgf("Failed to get Filter %s", key.String())
		return nil
	}

	return filter
}

// FilterRefToFilter converts a LocalFilterReference to a Filter.
func FilterRefToFilter(client cache.Cache, route client.Object, filterRef gwpav1alpha2.LocalFilterReference) *extv1alpha1.Filter {
	key := types.NamespacedName{
		Namespace: route.GetNamespace(),
		Name:      string(filterRef.Name),
	}
	filter := &extv1alpha1.Filter{}
	if err := client.Get(context.Background(), key, filter); err != nil {
		log.Error().Err(err).Msgf("Failed to get Filter %s", key.String())
		return nil
	}

	return filter
}
