package cache

import (
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ReferenceGrantsProcessor is responsible for processing ReferenceGrant objects
type ReferenceGrantsProcessor struct {
}

// Insert adds a ReferenceGrant to the cache and returns true if the route is effective
func (p *ReferenceGrantsProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	rg, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.referencegrants[utils.ObjectKey(rg)] = struct{}{}

	return true
}

// Delete removes a ReferenceGrant from the cache and returns true if the route was found
func (p *ReferenceGrantsProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	rg, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(rg)
	_, found := cache.referencegrants[key]
	delete(cache.referencegrants, key)

	return found
}
