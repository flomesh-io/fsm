package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type ServicesProcessor struct{}

func (p *ServicesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svc)
	cache.services[key] = struct{}{}

	return cache.isRoutableService(key)
}

func (p *ServicesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svc)
	_, found := cache.services[key]
	delete(cache.services, key)

	return found
}
