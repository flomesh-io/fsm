package cache

import (
	"sync"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// EndpointSlicesTrigger is responsible for processing EndpointSlices
type EndpointSlicesTrigger struct {
	mu sync.Mutex
}

// Insert adds the EndpointSlice object to the cache and returns true if the cache was modified
func (p *EndpointSlicesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	eps, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	if len(eps.Labels) == 0 {
		return false
	}

	svcName := eps.Labels[constants.KubernetesEndpointSliceServiceNameLabel]
	if len(svcName) == 0 {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	svcKey := client.ObjectKey{Namespace: eps.Namespace, Name: svcName}
	_, found := cache.endpointslices[svcKey]
	if !found {
		cache.endpointslices[svcKey] = make(map[client.ObjectKey]struct{})
	}
	cache.endpointslices[svcKey][utils.ObjectKey(eps)] = struct{}{}

	return cache.isRoutableService(svcKey)
}

// Delete removes the EndpointSlice object from the cache and returns true if the cache was modified
func (p *EndpointSlicesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	eps, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	owner := metav1.GetControllerOf(eps)
	if owner == nil {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	svcKey := client.ObjectKey{Namespace: eps.Namespace, Name: owner.Name}
	slices, found := cache.endpointslices[svcKey]
	if !found {
		return false
	}

	sliceKey := utils.ObjectKey(eps)
	_, found = slices[sliceKey]
	delete(cache.endpointslices[svcKey], sliceKey)

	if len(cache.endpointslices[svcKey]) == 0 {
		delete(cache.endpointslices, svcKey)
	}

	return found
}
