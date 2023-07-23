package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type GRPCRoutesProcessor struct {
}

func (p *GRPCRoutesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.GRPCRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.grpcroutes[utils.ObjectKey(route)] = struct{}{}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

func (p *GRPCRoutesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.GRPCRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(route)
	_, found := cache.grpcroutes[key]
	delete(cache.grpcroutes, key)

	return found
}
