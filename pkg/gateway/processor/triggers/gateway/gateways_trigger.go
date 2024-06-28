package gateway

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// GatewaysTrigger is responsible for processing Gateway objects
type GatewaysTrigger struct{}

// Insert adds the Gateway object to the cache and returns true if the cache was modified
func (p *GatewaysTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	gw, ok := obj.(*gwv1.Gateway)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	//key := utils.ObjectKey(gw)
	//
	//gw, err := cache.informers.GetListers().Gateway.Gateways(gw.NamespaceDerefOr).Get(gw.Name)
	//if err != nil {
	//	log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
	//	return false
	//}

	return utils.IsActiveGateway(gw)
}

// Delete removes the Gateway object from the cache and returns true if the cache was modified
func (p *GatewaysTrigger) Delete(obj interface{}, _ processor.Processor) bool {
	_, ok := obj.(*gwv1.Gateway)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return true
}
