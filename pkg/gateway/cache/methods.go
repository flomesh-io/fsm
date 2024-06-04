package cache

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/flomesh-io/fsm/pkg/k8s"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) getActiveGateways() []*gwv1.Gateway {
	class, err := gwutils.FindEffectiveGatewayClass(c.client)
	if err != nil {
		log.Error().Msgf("Failed to find GatewayClass: %v", err)
		return nil
	}

	list := &gwv1.GatewayList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ClassGatewayIndex, class.Name),
	}); err != nil {
		log.Error().Msgf("Failed to list Gateways: %v", err)
		return nil
	}

	return gwutils.FilterActiveGateways(gwutils.ToSlicePtr(list.Items))
}

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
