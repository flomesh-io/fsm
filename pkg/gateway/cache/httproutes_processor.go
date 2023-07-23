package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type HTTPRoutesProcessor struct {
}

func (p *HTTPRoutesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1beta1.HTTPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.httproutes[utils.ObjectKey(route)] = struct{}{}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

func (p *HTTPRoutesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1beta1.HTTPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(route)
	_, found := cache.httproutes[key]
	delete(cache.httproutes, key)

	return found
}
