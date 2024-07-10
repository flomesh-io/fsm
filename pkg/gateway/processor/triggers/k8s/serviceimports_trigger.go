package triggers

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// ServiceImportsTrigger is responsible for processing ServiceImport objects
type ServiceImportsTrigger struct{}

// Insert adds a ServiceImport to the processor and returns true if the route is effective
func (p *ServiceImportsTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	svcimp, ok := obj.(*mcsv1alpha1.ServiceImport)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableService(client.ObjectKeyFromObject(svcimp))
}

// Delete removes a ServiceImport from the processor and returns true if the route was found
func (p *ServiceImportsTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	svcimp, ok := obj.(*mcsv1alpha1.ServiceImport)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableService(client.ObjectKeyFromObject(svcimp))
}
