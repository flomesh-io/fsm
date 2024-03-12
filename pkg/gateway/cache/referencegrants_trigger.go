package cache

import (
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// ReferenceGrantTrigger is responsible for processing ReferenceGrant objects
type ReferenceGrantTrigger struct{}

// Insert adds a ReferenceGrant to the cache and returns true if the ReferenceGrant is effective
func (p *ReferenceGrantTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	_, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	// TODO: implement the insert logic
	return true
}

// Delete removes a ReferenceGrant from the cache and returns true if the ReferenceGrant was found
func (p *ReferenceGrantTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	_, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	// TODO: implement the delete logic
	return true
}
