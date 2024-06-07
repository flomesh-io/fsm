package cache

import (
	"context"

	"github.com/flomesh-io/fsm/pkg/k8s"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *GatewayCache) getSecretFromCache(key client.ObjectKey) (*corev1.Secret, error) {
	obj := &corev1.Secret{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *GatewayCache) getConfigMapFromCache(key client.ObjectKey) (*corev1.ConfigMap, error) {
	obj := &corev1.ConfigMap{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *GatewayCache) getServiceFromCache(key client.ObjectKey) (*corev1.Service, error) {
	obj := &corev1.Service{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *GatewayCache) isHeadlessService(key client.ObjectKey) bool {
	service, err := c.getServiceFromCache(key)
	if err != nil {
		log.Error().Msgf("failed to get service from cache: %v", err)
		return false
	}

	return k8s.IsHeadlessService(*service)
}
