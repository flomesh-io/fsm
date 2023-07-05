package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	"k8s.io/klog/v2"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type TCPRoutesProcessor struct {
}

func (p *TCPRoutesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	cache.tcproutes[utils.ObjectKey(route)] = struct{}{}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

func (p *TCPRoutesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(route)
	_, found := cache.tcproutes[key]
	delete(cache.tcproutes, key)

	return found
}
