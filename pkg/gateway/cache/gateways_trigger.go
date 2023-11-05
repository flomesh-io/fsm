package cache

import (
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// GatewaysTrigger is responsible for processing Gateway objects
type GatewaysTrigger struct {
}

// Insert adds the Gateway object to the cache and returns true if the cache was modified
func (p *GatewaysTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	gw, ok := obj.(*gwv1beta1.Gateway)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(gw)
	//if err := cache.client.Get(context.TODO(), key, gw); err != nil {
	//	log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
	//	return false
	//}
	gw, err := cache.informers.GetListers().Gateway.Gateways(gw.Namespace).Get(gw.Name)
	//obj, exists, err := cache.informers.GetByKey(informers.InformerKeyGatewayAPIGateway, key.String())
	if err != nil {
		log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
		return false
	}
	//if !exists {
	//	log.Error().Msgf("Gateway %s doesn't exist", key)
	//	return false
	//}
	//
	//gw = obj.(*gwv1beta1.Gateway)
	if utils.IsActiveGateway(gw) {
		cache.gateways[gw.Namespace] = utils.ObjectKey(gw)
		return true
	}

	return false
}

// Delete removes the Gateway object from the cache and returns true if the cache was modified
func (p *GatewaysTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	gw, ok := obj.(*gwv1beta1.Gateway)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := gw.Namespace
	_, found := cache.gateways[key]
	delete(cache.gateways, key)

	return found
}
