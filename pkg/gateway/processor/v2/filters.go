package v2

import (
	"context"

	"k8s.io/utils/ptr"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
)

func (c *ConfigGenerator) resolveFilterDefinition(filterType extv1alpha1.FilterType, filterScope extv1alpha1.FilterScope, ref *gwv1.LocalObjectReference) *extv1alpha1.FilterDefinition {
	if ref == nil {
		return nil
	}

	definition := &extv1alpha1.FilterDefinition{}
	if err := c.client.Get(context.Background(), types.NamespacedName{Name: string(ref.Name)}, definition); err != nil {
		log.Error().Msgf("Failed to resolve FilterDefinition: %s", err)
		return nil
	}

	if filterType != definition.Spec.Type {
		log.Error().Msgf("FilterDefinition %s is not of type %s", definition.Name, filterType)
		return nil
	}

	definitionScope := ptr.Deref(definition.Spec.Scope, extv1alpha1.FilterScopeRoute)
	if filterScope != definitionScope {
		log.Error().Msgf("FilterDefinition %s is not of scope %s", definition.Name, filterScope)
		return nil
	}

	return definition
}

//gocyclo:ignore
func (c *ConfigGenerator) resolveFilterConfig(ns string, ref *gwv1.LocalObjectReference) map[string]interface{} {
	if ref == nil {
		return map[string]interface{}{}
	}

	key := types.NamespacedName{Namespace: ns, Name: string(ref.Name)}
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
	case constants.GatewayHTTPLogKind:
		obj := &extv1alpha1.HTTPLog{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve HTTPLog: %s", err)
			return map[string]interface{}{}
		}

		l2 := fgwv2.HTTPLogSpec{}
		if err := gwutils.DeepCopy(&l2, &obj.Spec); err != nil {
			log.Error().Msgf("Failed to copy HTTPLog: %s", err)
			return map[string]interface{}{}
		}

		return toMap("httpLog", &l2)
	case constants.GatewayMetricsKind:
		obj := &extv1alpha1.Metrics{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve Metrics: %s", err)
			return map[string]interface{}{}
		}

		m2 := fgwv2.MetricsSpec{}
		if err := gwutils.DeepCopy(&m2, &obj.Spec); err != nil {
			log.Error().Msgf("Failed to copy Metrics: %s", err)
			return map[string]interface{}{}
		}

		return toMap("metrics", &m2)
	case constants.GatewayZipkinKind:
		obj := &extv1alpha1.Zipkin{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve Zipkin: %s", err)
			return map[string]interface{}{}
		}

		return toMap("zipkin", &obj.Spec)
	case constants.GatewayProxyTagKind:
		obj := &extv1alpha1.ProxyTag{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve ProxyTag: %s", err)
			return map[string]interface{}{}
		}

		return toMap("proxyTag", &obj.Spec)
	case constants.GatewayIPRestrictionKind:
		obj := &extv1alpha1.IPRestriction{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve IPRestriction: %s", err)
			return map[string]interface{}{}
		}

		return toMap("ipRestriction", &obj.Spec)
	case constants.GatewayExternalRateLimitKind:
		obj := &extv1alpha1.ExternalRateLimit{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve ExternalRateLimit: %s", err)
			return map[string]interface{}{}
		}

		return toMap("externalRateLimit", &obj.Spec)
	case constants.GatewayRequestTerminationKind:
		obj := &extv1alpha1.RequestTermination{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve RequestTermination: %s", err)
			return map[string]interface{}{}
		}

		return toMap("requestTermination", &obj.Spec)
	case constants.GatewayConcurrencyLimitKind:
		obj := &extv1alpha1.ConcurrencyLimit{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve ConcurrencyLimit: %s", err)
			return map[string]interface{}{}
		}

		return toMap("concurrencyLimit", &obj.Spec)
	case constants.GatewayDNSModifierKind:
		obj := &extv1alpha1.DNSModifier{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve DNSModifier: %s", err)
			return map[string]interface{}{}
		}

		return toMap("dnsModifier", &obj.Spec)
	case constants.GatewayAPIExtensionFilterConfigKind:
		obj := &extv1alpha1.FilterConfig{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("Failed to resolve FilterConfig: %s", err)
			return map[string]interface{}{}
		}

		vals := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(obj.Spec.Config), &vals); err != nil {
			log.Error().Msgf("Failed to unmarshal FilterConfig: %s", err)
			return map[string]interface{}{}
		}

		return vals
	}

	return map[string]interface{}{}
}

//func toFilterProtocol(protocol gwv1.ProtocolType) *extv1alpha1.FilterProtocol {
//	switch protocol {
//	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
//		return ptr.To(extv1alpha1.FilterProtocolHTTP)
//	case gwv1.TCPProtocolType:
//		return ptr.To(extv1alpha1.FilterProtocolTCP)
//	case gwv1.UDPProtocolType:
//		return ptr.To(extv1alpha1.FilterProtocolUDP)
//	default:
//		return nil
//	}
//}

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
