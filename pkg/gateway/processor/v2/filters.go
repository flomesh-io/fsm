package v2

import (
	"context"

	"sigs.k8s.io/yaml"

	ghodssyaml "github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
)

func (c *ConfigGenerator) resolveFilterDefinition(ref gwv1.LocalObjectReference) *extv1alpha1.FilterDefinition {
	definition := &extv1alpha1.FilterDefinition{}
	if err := c.client.Get(context.Background(), types.NamespacedName{Name: string(ref.Name)}, definition); err != nil {
		log.Error().Msgf("Failed to resolve FilterDefinition: %s", err)
		return nil
	}

	return definition
}

func (c *ConfigGenerator) resolveFilterConfig(ref gwv1.LocalObjectReference) map[string]interface{} {
	key := types.NamespacedName{Namespace: c.gateway.Namespace, Name: string(ref.Name)}
	ctx := context.Background()

	switch ref.Kind {
	case constants.CircuitBreakerKind:
		obj := &extv1alpha1.CircuitBreaker{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve CircuitBreaker: %s", err)
			return map[string]interface{}{}
		}

		return toMap(&obj.Spec)
	case constants.FaultInjectionKind:
		obj := &extv1alpha1.FaultInjection{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve FaultInjection: %s", err)
			return map[string]interface{}{}
		}

		return toMap(&obj.Spec)
	}

	return map[string]interface{}{}
}

func toFilterProtocol(protocol gwv1.ProtocolType) *extv1alpha1.FilterProtocol {
	switch protocol {
	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
		return ptr.To(extv1alpha1.FilterProtocolHTTP)
	case gwv1.TCPProtocolType:
		return ptr.To(extv1alpha1.FilterProtocolTCP)
	case gwv1.UDPProtocolType:
		return ptr.To(extv1alpha1.FilterProtocolUDP)
	default:
		return nil
	}
}

func toMap(spec interface{}) map[string]interface{} {
	bytes, err := ghodssyaml.Marshal(spec)
	if err != nil {
		log.Error().Msgf("Failed to marshal spec: %v", err)
		return map[string]interface{}{}
	}

	vals := map[string]interface{}{}
	err = yaml.Unmarshal(bytes, &vals)
	if err != nil {
		log.Error().Msgf("Failed to read values: %v", err)
		return map[string]interface{}{}
	}

	if len(vals) == 0 {
		return map[string]interface{}{}
	}

	return vals
}
