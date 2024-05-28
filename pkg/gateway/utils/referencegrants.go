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

func getGatewayRefGrants(c cache.Cache) []*gwv1beta1.ReferenceGrant {
	gatewayRefGrantList := &gwv1beta1.ReferenceGrantList{}
	if err := c.List(context.Background(), gatewayRefGrantList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.GatewayAPIGatewayKind),
	}); err != nil {
		return nil
	}

	return ToSlicePtr(gatewayRefGrantList.Items)
}

func getHostnameRefGrants(c cache.Cache) []*gwv1beta1.ReferenceGrant {
	list := &gwv1beta1.ReferenceGrantList{}
	if err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: gwtypes.OrSelectors(
			fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.GatewayAPIHTTPRouteKind),
			fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.GatewayAPIGRPCRouteKind),
		),
	}); err != nil {
		return nil
	}

	return ToSlicePtr(list.Items)
}

func getHTTPRouteRefGrants(c cache.Cache) []*gwv1beta1.ReferenceGrant {
	httpRouteRefGrantList := &gwv1beta1.ReferenceGrantList{}
	if err := c.List(context.Background(), httpRouteRefGrantList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.GatewayAPIHTTPRouteKind),
	}); err != nil {
		return nil
	}

	return ToSlicePtr(httpRouteRefGrantList.Items)
}

func getGRPCRouteRefGrants(c cache.Cache) []*gwv1beta1.ReferenceGrant {
	grpcRouteRefGrantList := &gwv1beta1.ReferenceGrantList{}
	if err := c.List(context.Background(), grpcRouteRefGrantList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.GatewayAPIGRPCRouteKind),
	}); err != nil {
		return nil
	}

	return ToSlicePtr(grpcRouteRefGrantList.Items)
}

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

func GetSecretRefGrants(c cache.Cache) []*gwv1beta1.ReferenceGrant {
	list := &gwv1beta1.ReferenceGrantList{}
	err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.TargetKindRefGrantIndex, constants.KubernetesSecretKind),
	})
	if err != nil {
		log.Error().Msgf("Failed to list ReferenceGrants: %v", err)
		return nil
	}

	return ToSlicePtr(list.Items)
}

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
