package cache

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ConfigMapTrigger is a processor for ConfigMap objects
type ConfigMapTrigger struct{}

// Insert adds a ConfigMap object to the cache and returns true if the cache is changed
func (p *ConfigMapTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(cm)

	return cache.isConfigMapReferred(key)
}

// Delete removes a ConfigMap object from the cache and returns true if the cache is changed
func (p *ConfigMapTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(cm)

	return cache.isConfigMapReferred(key)
}
