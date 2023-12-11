package cache

import (
	"sync"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// GatewayClassesTrigger is responsible for processing GatewayClass objects
type GatewayClassesTrigger struct {
	mu sync.Mutex
}

// Insert adds the GatewayClass object to the cache and returns true if the cache was modified
func (p *GatewayClassesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	class, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := class.GetName()
	//if err := cache.client.Get(context.TODO(), key, class); err != nil {
	//	log.Error().Msgf("Failed to get GatewayClass %s: %s", key, err)
	//	return false
	//}
	class, err := cache.informers.GetListers().GatewayClass.Get(key)
	//obj, exists, err := cache.informers.GetByKey(informers.InformerKeyGatewayAPIGatewayClass, key)
	if err != nil {
		log.Error().Msgf("Failed to get GatewayClass %s: %s", key, err)
		return false
	}
	//if !exists {
	//	log.Error().Msgf("GatewayClass %s doesn't exist", key)
	//	return false
	//}

	//class = obj.(*gwv1beta1.GatewayClass)
	if utils.IsEffectiveGatewayClass(class) {
		p.mu.Lock()
		defer p.mu.Unlock()

		cache.gatewayclass = class
		return true
	}

	return false
}

// Delete removes the GatewayClass object from the cache and returns true if the cache was modified
func (p *GatewayClassesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	class, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if cache.gatewayclass == nil {
		return false
	}

	if class.Name == cache.gatewayclass.Name {
		cache.gatewayclass = nil
		return true
	}

	return false
}