package gateway

import "github.com/flomesh-io/fsm/pkg/gateway/processor"

// GatewayClassesTrigger is responsible for processing GatewayClass objects
type GatewayClassesTrigger struct{}

// Insert adds the GatewayClass object to the processor and returns true if the processor was modified
func (p *GatewayClassesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	return true
}

// Delete removes the GatewayClass object from the processor and returns true if the processor was modified
func (p *GatewayClassesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	return true
}
