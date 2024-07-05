package utils

import (
	"context"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
)

func ExtensionRefToFilter(client cache.Cache, route client.Object, extensionRef *gwv1.LocalObjectReference) *fgwv2.Filter {
	filter := &extv1alpha1.Filter{}
	if err := client.Get(context.Background(), types.NamespacedName{
		Namespace: route.GetNamespace(),
		Name:      string(extensionRef.Name),
	}, filter); err != nil {
		return nil
	}

	if len(filter.Spec.Scripts) == 1 {
		for key, value := range filter.Spec.Scripts {
			return &fgwv2.Filter{
				Type:   filter.Spec.Type,
				Name:   key,
				Script: value,
			}
		}
	}

	return nil
}
