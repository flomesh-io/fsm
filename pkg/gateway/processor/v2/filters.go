package v2

import (
	"context"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"sigs.k8s.io/yaml"

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

func (c *ConfigGenerator) resolveFilterConfig(ref *gwv1.LocalObjectReference) map[string]interface{} {
	if ref == nil {
		return map[string]interface{}{}
	}

	key := types.NamespacedName{Namespace: c.gateway.Namespace, Name: string(ref.Name)}
	ctx := context.Background()

	switch ref.Kind {
	case constants.CircuitBreakerKind:
		obj := &extv1alpha1.CircuitBreaker{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve CircuitBreaker: %s", err)
			return map[string]interface{}{}
		}

		c2 := fgwv2.CircuitBreakerSpec{}
		if err := gwutils.DeepCopy(&c2, &obj.Spec); err != nil {
			log.Error().Msgf("Failed to copy CircuitBreaker: %s", err)
			return map[string]interface{}{}
		}

		return toMap("circuitBreak", &c2)
	case constants.FaultInjectionKind:
		obj := &extv1alpha1.FaultInjection{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve FaultInjection: %s", err)
			return map[string]interface{}{}
		}

		f2 := fgwv2.FaultInjectionSpec{}
		if err := gwutils.DeepCopy(&f2, &obj.Spec); err != nil {
			log.Error().Msgf("Failed to copy FaultInjection: %s", err)
			return map[string]interface{}{}
		}

		return toMap("faultInjection", &f2)
	case constants.RateLimitKind:
		obj := &extv1alpha1.RateLimit{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve RateLimit: %s", err)
			return map[string]interface{}{}
		}

		r2 := fgwv2.RateLimitSpec{}
		if err := gwutils.DeepCopy(&r2, &obj.Spec); err != nil {
			log.Error().Msgf("Failed to copy RateLimit: %s", err)
			return map[string]interface{}{}
		}

		return toMap("rateLimit", &r2)
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

func toMap(key string, spec interface{}) map[string]interface{} {
	bytes, err := yaml.Marshal(spec)
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

	return map[string]interface{}{
		key: vals,
	}
}
