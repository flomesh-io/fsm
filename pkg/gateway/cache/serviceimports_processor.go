package cache

import (
	svcimpv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/serviceimport/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	"k8s.io/klog/v2"
)

type ServiceImportsProcessor struct {
}

func (p *ServiceImportsProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	svcimp, ok := obj.(*svcimpv1alpha1.ServiceImport)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svcimp)
	cache.serviceimports[key] = struct{}{}

	return cache.isRoutableService(key)
}

func (p *ServiceImportsProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	svcimp, ok := obj.(*svcimpv1alpha1.ServiceImport)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(svcimp)
	_, found := cache.serviceimports[key]
	delete(cache.serviceimports, key)

	return found
}
