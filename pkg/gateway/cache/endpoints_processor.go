package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type EndpointsProcessor struct {
}

func (p *EndpointsProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(ep)
	cache.endpoints[key] = struct{}{}

	return cache.isRoutableService(key)
}

func (p *EndpointsProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(ep)
	_, found := cache.endpoints[key]
	delete(cache.endpoints, key)

	return found
}
