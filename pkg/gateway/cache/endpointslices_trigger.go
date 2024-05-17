package cache

import (
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EndpointSlicesTrigger is responsible for processing EndpointSlices
type EndpointSlicesTrigger struct{}

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

	svcName := eps.Labels[discoveryv1.LabelServiceName]
	if len(svcName) == 0 {
		return false
	}

	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	svcKey := client.ObjectKey{Namespace: eps.Namespace, Name: svcName}
	//_, found := cache.endpointslices[svcKey]
	//if !found {
	//	cache.endpointslices[svcKey] = make(map[client.ObjectKey]struct{})
	//}
	//cache.endpointslices[svcKey][utils.ObjectKey(eps)] = struct{}{}

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

	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	svcKey := client.ObjectKey{Namespace: eps.Namespace, Name: owner.Name}
	//slices, found := cache.endpointslices[svcKey]
	//if !found {
	//	return false
	//}
	//
	//sliceKey := utils.ObjectKey(eps)
	//_, found = slices[sliceKey]
	//delete(cache.endpointslices[svcKey], sliceKey)
	//
	//if len(cache.endpointslices[svcKey]) == 0 {
	//	delete(cache.endpointslices, svcKey)
	//}
	//
	//return found

	return cache.isRoutableService(svcKey)
}
