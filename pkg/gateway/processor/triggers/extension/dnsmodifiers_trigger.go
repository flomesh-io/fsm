package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// DNSModifierTrigger is a processor for DNSModifier objects
type DNSModifierTrigger struct{}

// Insert adds a DNSModifier object to the processor and returns true if the processor is changed
func (p *DNSModifierTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.DNSModifier)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}

// Delete removes a DNSModifier object from the processor and returns true if the processor is changed
func (p *DNSModifierTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.DNSModifier)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}
