package cache

import (
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ServiceImportsProcessor is responsible for processing ServiceImport objects
type ServiceImportsProcessor struct {
}

// Insert adds a ServiceImport to the cache and returns true if the route is effective
func (p *ServiceImportsProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	svcimp, ok := obj.(*mcsv1alpha1.ServiceImport)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svcimp)
	cache.serviceimports[key] = struct{}{}

	return cache.isRoutableService(key)
}

// Delete removes a ServiceImport from the cache and returns true if the route was found
func (p *ServiceImportsProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	svcimp, ok := obj.(*mcsv1alpha1.ServiceImport)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svcimp)
	_, found := cache.serviceimports[key]
	delete(cache.serviceimports, key)

	return found
}
