package gateway

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// GatewaysTrigger is responsible for processing Gateway objects
type GatewaysTrigger struct{}

// Insert adds the Gateway object to the processor and returns true if the processor was modified
func (p *GatewaysTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	gw, ok := obj.(*gwv1.Gateway)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return utils.IsAcceptedGateway(gw)
}

// Delete removes the Gateway object from the processor and returns true if the processor was modified
func (p *GatewaysTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	gw, ok := obj.(*gwv1.Gateway)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.OnDeleteGateway(gw)
}
