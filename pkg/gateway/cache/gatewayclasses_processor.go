package cache

import (
	"context"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	"k8s.io/klog/v2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewayClassesProcessor struct {
}

func (p *GatewayClassesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	class, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(class)
	if err := cache.client.Get(context.TODO(), key, class); err != nil {
		klog.Errorf("Failed to get GatewayClass %s: %s", key, err)
		return false
	}

	if utils.IsEffectiveGatewayClass(class) {
		cache.gatewayclass = class
		return true
	}

	return false
}

func (p *GatewayClassesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	class, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		klog.Errorf("unexpected object type %T", obj)
		return false
	}

	if cache.gatewayclass == nil {
		return false
	}

	if class.Name == cache.gatewayclass.Name {
		cache.gatewayclass = nil
		return true
	}

	return false
}
