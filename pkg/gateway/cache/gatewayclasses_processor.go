package cache

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// GatewayClassesProcessor is responsible for processing GatewayClass objects
type GatewayClassesProcessor struct {
}

// Insert adds the GatewayClass object to the cache and returns true if the cache was modified
func (p *GatewayClassesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	class, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := client.ObjectKey{Name: class.GetName()}
	//if err := cache.client.Get(context.TODO(), key, class); err != nil {
	//	log.Error().Msgf("Failed to get GatewayClass %s: %s", key, err)
	//	return false
	//}
	obj, exists, err := cache.informers.GetByKey(informers.InformerKeyGatewayAPIGatewayClass, key.String())
	if err != nil {
		log.Error().Msgf("Failed to get GatewayClass %s: %s", key, err)
		return false
	}
	if !exists {
		log.Error().Msgf("GatewayClass %s doesn't exist", key)
		return false
	}

	class = obj.(*gwv1beta1.GatewayClass)
	if utils.IsEffectiveGatewayClass(class) {
		cache.gatewayclass = class
		return true
	}

	return false
}

// Delete removes the GatewayClass object from the cache and returns true if the cache was modified
func (p *GatewayClassesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	class, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	if cache.gatewayclass == nil {
		return false
	}

	if class.Name == cache.gatewayclass.Name {
		cache.gatewayclass = nil
		return true
	}

	return false
}
