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
		log.Error().Msgf("[GW] Failed to resolve FilterDefinition: %s", err)
		return nil
	}

	if filterType != definition.Spec.Type {
		log.Error().Msgf("[GW] FilterDefinition %s is not of type %s", definition.Name, filterType)
		return nil
	}

	definitionScope := ptr.Deref(definition.Spec.Scope, extv1alpha1.FilterScopeRoute)
	if filterScope != definitionScope {
		log.Error().Msgf("[GW] FilterDefinition %s is not of scope %s", definition.Name, filterScope)
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
		k := "circuitBreak"

		obj := &extv1alpha1.CircuitBreaker{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve CircuitBreaker: %s", err)
			return emptyConfig(k)
		}

		c2 := fgwv2.CircuitBreakerSpec{}
		if err := gwutils.DeepCopy(&c2, &obj.Spec); err != nil {
			log.Error().Msgf("[GW] Failed to copy CircuitBreaker: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &c2)
	case constants.FaultInjectionKind:
		k := "faultInjection"

		obj := &extv1alpha1.FaultInjection{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve FaultInjection: %s", err)
			return emptyConfig(k)
		}

		f2 := fgwv2.FaultInjectionSpec{}
		if err := gwutils.DeepCopy(&f2, &obj.Spec); err != nil {
			log.Error().Msgf("[GW] Failed to copy FaultInjection: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &f2)
	case constants.RateLimitKind:
		k := "rateLimit"

		obj := &extv1alpha1.RateLimit{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve RateLimit: %s", err)
			return emptyConfig(k)
		}

		r2 := fgwv2.RateLimitSpec{}
		if err := gwutils.DeepCopy(&r2, &obj.Spec); err != nil {
			log.Error().Msgf("[GW] Failed to copy RateLimit: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &r2)
	case constants.GatewayHTTPLogKind:
		k := "httpLog"

		obj := &extv1alpha1.HTTPLog{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve HTTPLog: %s", err)
			return emptyConfig(k)
		}

		l2 := fgwv2.HTTPLogSpec{}
		if err := gwutils.DeepCopy(&l2, &obj.Spec); err != nil {
			log.Error().Msgf("[GW] Failed to copy HTTPLog: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &l2)
	case constants.GatewayMetricsKind:
		k := "metrics"

		obj := &extv1alpha1.Metrics{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve Metrics: %s", err)
			return emptyConfig(k)
		}

		m2 := fgwv2.MetricsSpec{}
		if err := gwutils.DeepCopy(&m2, &obj.Spec); err != nil {
			log.Error().Msgf("[GW] Failed to copy Metrics: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &m2)
	case constants.GatewayZipkinKind:
		k := "zipkin"

		obj := &extv1alpha1.Zipkin{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve Zipkin: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &obj.Spec)
	case constants.GatewayProxyTagKind:
		k := "proxyTag"

		obj := &extv1alpha1.ProxyTag{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve ProxyTag: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &obj.Spec)
	case constants.GatewayIPRestrictionKind:
		k := "ipRestriction"

		obj := &extv1alpha1.IPRestriction{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve IPRestriction: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &obj.Spec)
	case constants.GatewayExternalRateLimitKind:
		k := "externalRateLimit"

		obj := &extv1alpha1.ExternalRateLimit{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve ExternalRateLimit: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &obj.Spec)
	case constants.GatewayRequestTerminationKind:
		k := "requestTermination"

		obj := &extv1alpha1.RequestTermination{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve RequestTermination: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &obj.Spec)
	case constants.GatewayConcurrencyLimitKind:
		k := "concurrencyLimit"

		obj := &extv1alpha1.ConcurrencyLimit{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve ConcurrencyLimit: %s", err)
			return emptyConfig(k)
		}

		return toMap(k, &obj.Spec)
	case constants.GatewayDNSModifierKind:
		k := "dnsModifier"

		obj := &extv1alpha1.DNSModifier{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve DNSModifier: %s", err)
			return emptyConfig(k)
		}

		result := struct {
			Domains []extv1alpha1.DNSDomain `json:"domains,omitempty"`
		}{
			Domains: []extv1alpha1.DNSDomain{},
		}

		for _, zone := range obj.Spec.Zones {
			result.Domains = append(result.Domains, zone.Domains...)
		}

		return toMap(k, &result)
	case constants.GatewayAPIExtensionFilterConfigKind:
		obj := &extv1alpha1.FilterConfig{}
		if err := c.client.Get(ctx, key, obj); err != nil {
			log.Error().Msgf("[GW] Failed to resolve FilterConfig: %s", err)
			return map[string]interface{}{}
		}

		vals := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(obj.Spec.Config), &vals); err != nil {
			log.Error().Msgf("[GW] Failed to unmarshal FilterConfig: %s", err)
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
		log.Error().Msgf("[GW] Failed to marshal spec: %v", err)
		return emptyConfig(key)
	}

	vals := map[string]interface{}{}
	err = yaml.Unmarshal(bytes, &vals)
	if err != nil {
		log.Error().Msgf("[GW] Failed to read values: %v", err)
		return emptyConfig(key)
	}

	if len(vals) == 0 {
		return emptyConfig(key)
	}

	return map[string]interface{}{
		key: vals,
	}
}

func emptyConfig(key string) map[string]interface{} {
	return map[string]interface{}{
		key: map[string]interface{}{},
	}
}
