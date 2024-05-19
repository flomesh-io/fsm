package cache

import (
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ServiceImportsTrigger is responsible for processing ServiceImport objects
type ServiceImportsTrigger struct{}

// Insert adds a ServiceImport to the cache and returns true if the route is effective
func (p *ServiceImportsTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	svcimp, ok := obj.(*mcsv1alpha1.ServiceImport)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svcimp)

	return cache.isRoutableService(key)
}

// Delete removes a ServiceImport from the cache and returns true if the route was found
func (p *ServiceImportsTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	svcimp, ok := obj.(*mcsv1alpha1.ServiceImport)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svcimp)

	return cache.isRoutableService(key)
}
