package cache

import (
	"context"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewaysProcessor struct {
}

func (p *GatewaysProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	gw, ok := obj.(*gwv1beta1.Gateway)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(gw)
	if err := cache.client.Get(context.TODO(), key, gw); err != nil {
		log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
		return false
	}

	if utils.IsActiveGateway(gw) {
		cache.gateways[gw.Namespace] = utils.ObjectKey(gw)
		return true
	}

	return false
}

func (p *GatewaysProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
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
