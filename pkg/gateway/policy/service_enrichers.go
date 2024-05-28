package policy

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/circuitbreaking"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/healthcheck"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/loadbalancer"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/retry"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/sessionsticky"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/upstreamtls"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

type ResolveServicePortNameFunc func(referer client.Object, ref gwv1alpha2.NamespacedPolicyTargetReference, port int32) *fgw.ServicePortName
type ResolveSecretFunc func(referer client.Object, ref gwv1.SecretObjectReference) (*corev1.Secret, error)

// ServicePolicyEnricher is an interface for enriching service level policies
type ServicePolicyEnricher interface {
	// Enrich enriches the service config with the service level policy based on the service port name
	Enrich(svcPortName string, svcCfg *fgw.ServiceConfig)
}

// ---

func NewSessionStickyPolicyEnricher(cache cache.Cache, selector fields.Selector, targetRefToServicePortName ResolveServicePortNameFunc) ServicePolicyEnricher {
	configs := make(map[string]*gwpav1alpha1.SessionStickyConfig)

	for _, sessionSticky := range gwutils.SortResources(gwutils.GetSessionStickies(cache, selector)) {
		sessionSticky := sessionSticky.(*gwpav1alpha1.SessionStickyPolicy)

		for _, p := range sessionSticky.Spec.Ports {
			if svcPortName := targetRefToServicePortName(sessionSticky, sessionSticky.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
				cfg := sessionsticky.ComputeSessionStickyConfig(p.Config, sessionSticky.Spec.DefaultConfig)

				if cfg == nil {
					continue
				}

				if _, ok := configs[svcPortName.String()]; ok {
					log.Warn().Msgf("Policy is already defined for service port %s, SessionStickyPolicy %s/%s:%d will be dropped", svcPortName.String(), sessionSticky.Namespace, sessionSticky.Name, p.Port)
					continue
				}

				configs[svcPortName.String()] = cfg
			}
		}
	}

	return &sessionStickyPolicyEnricher{
		data: configs,
	}
}

// sessionStickyPolicyEnricher is an enricher for session sticky policies
type sessionStickyPolicyEnricher struct {
	data map[string]*gwpav1alpha1.SessionStickyConfig
}

func (e *sessionStickyPolicyEnricher) Enrich(svcPortName string, svcCfg *fgw.ServiceConfig) {
	if len(e.data) == 0 {
		return
	}

	if ssCfg, exists := e.data[svcPortName]; exists {
		svcCfg.StickyCookieName = ssCfg.CookieName
		svcCfg.StickyCookieExpires = ssCfg.Expires
	}
}

// ---

func NewLoadBalancerPolicyEnricher(cache cache.Cache, selector fields.Selector, targetRefToServicePortName ResolveServicePortNameFunc) ServicePolicyEnricher {
	loadBalancers := make(map[string]*gwpav1alpha1.LoadBalancerType)

	for _, lb := range gwutils.SortResources(gwutils.GetLoadBalancers(cache, selector)) {
		lb := lb.(*gwpav1alpha1.LoadBalancerPolicy)

		for _, p := range lb.Spec.Ports {
			if svcPortName := targetRefToServicePortName(lb, lb.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
				t := loadbalancer.ComputeLoadBalancerType(p.Type, lb.Spec.DefaultType)

				if t == nil {
					continue
				}

				if _, ok := loadBalancers[svcPortName.String()]; ok {
					log.Warn().Msgf("Policy is already defined for service port %s, LoadBalancerPolicy %s/%s:%d will be dropped", svcPortName.String(), lb.Namespace, lb.Name, p.Port)
					continue
				}

				loadBalancers[svcPortName.String()] = t
			}
		}
	}

	return &LoadBalancerPolicyEnricher{
		data: loadBalancers,
	}
}

// LoadBalancerPolicyEnricher is an enricher for load balancer policies
type LoadBalancerPolicyEnricher struct {
	data map[string]*gwpav1alpha1.LoadBalancerType
}

func (e *LoadBalancerPolicyEnricher) Enrich(svcPortName string, svcCfg *fgw.ServiceConfig) {
	if len(e.data) == 0 {
		return
	}

	if lbType, exists := e.data[svcPortName]; exists {
		svcCfg.LoadBalancer = lbType
	}
}

// ---

func NewCircuitBreakingPolicyEnricher(cache cache.Cache, selector fields.Selector, targetRefToServicePortName ResolveServicePortNameFunc) ServicePolicyEnricher {
	configs := make(map[string]*gwpav1alpha1.CircuitBreakingConfig)

	for _, cb := range gwutils.SortResources(gwutils.GetCircuitBreakings(cache, selector)) {
		cb := cb.(*gwpav1alpha1.CircuitBreakingPolicy)

		for _, p := range cb.Spec.Ports {
			if svcPortName := targetRefToServicePortName(cb, cb.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
				cfg := circuitbreaking.ComputeCircuitBreakingConfig(p.Config, cb.Spec.DefaultConfig)

				if cfg == nil {
					continue
				}

				if _, ok := configs[svcPortName.String()]; ok {
					log.Warn().Msgf("Policy is already defined for service port %s, CircuitBreakingPolicy %s/%s:%d will be dropped", svcPortName.String(), cb.Namespace, cb.Name, p.Port)
					continue
				}

				configs[svcPortName.String()] = cfg
			}
		}
	}

	return &CircuitBreakingPolicyEnricher{
		data: configs,
	}
}

// CircuitBreakingPolicyEnricher is an enricher for circuit breaking policies
type CircuitBreakingPolicyEnricher struct {
	data map[string]*gwpav1alpha1.CircuitBreakingConfig
}

func (e *CircuitBreakingPolicyEnricher) Enrich(svcPortName string, svcCfg *fgw.ServiceConfig) {
	if len(e.data) == 0 {
		return
	}

	if cbCfg, exists := e.data[svcPortName]; exists {
		svcCfg.CircuitBreaking = newCircuitBreaking(cbCfg)
	}
}

// ---

func NewHealthCheckPolicyEnricher(cache cache.Cache, selector fields.Selector, targetRefToServicePortName ResolveServicePortNameFunc) ServicePolicyEnricher {
	configs := make(map[string]*gwpav1alpha1.HealthCheckConfig)

	for _, hc := range gwutils.SortResources(gwutils.GetHealthChecks(cache, selector)) {
		hc := hc.(*gwpav1alpha1.HealthCheckPolicy)

		for _, p := range hc.Spec.Ports {
			if svcPortName := targetRefToServicePortName(hc, hc.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
				cfg := healthcheck.ComputeHealthCheckConfig(p.Config, hc.Spec.DefaultConfig)

				if cfg == nil {
					continue
				}

				if _, ok := configs[svcPortName.String()]; ok {
					log.Warn().Msgf("Policy is already defined for service port %s, HealthCheckPolicy %s/%s:%d will be dropped", svcPortName.String(), hc.Namespace, hc.Name, p.Port)
					continue
				}

				configs[svcPortName.String()] = cfg
			}
		}
	}

	return &HealthCheckPolicyEnricher{
		data: configs,
	}
}

// HealthCheckPolicyEnricher is an enricher for health check policies
type HealthCheckPolicyEnricher struct {
	data map[string]*gwpav1alpha1.HealthCheckConfig
}

func (e *HealthCheckPolicyEnricher) Enrich(svcPortName string, svcCfg *fgw.ServiceConfig) {
	if len(e.data) == 0 {
		return
	}

	if hcCfg, exists := e.data[svcPortName]; exists {
		svcCfg.HealthCheck = newHealthCheck(hcCfg)
	}
}

// ---

func NewUpstreamTLSPolicyEnricher(cache cache.Cache, selector fields.Selector, targetRefToServicePortName ResolveServicePortNameFunc, secretRefToSecret ResolveSecretFunc) ServicePolicyEnricher {
	configs := make(map[string]*UpstreamTLSConfig)

	for _, upstreamTLS := range gwutils.SortResources(gwutils.GetUpStreamTLSes(cache, selector)) {
		upstreamTLS := upstreamTLS.(*gwpav1alpha1.UpstreamTLSPolicy)

		for _, p := range upstreamTLS.Spec.Ports {
			if svcPortName := targetRefToServicePortName(upstreamTLS, upstreamTLS.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
				cfg := upstreamtls.ComputeUpstreamTLSConfig(p.Config, upstreamTLS.Spec.DefaultConfig)

				if cfg == nil {
					continue
				}

				secret, err := secretRefToSecret(upstreamTLS, cfg.CertificateRef)
				if err != nil {
					log.Error().Msgf("Failed to resolve Secret: %s", err)
					continue
				}

				if _, ok := configs[svcPortName.String()]; ok {
					log.Warn().Msgf("Policy is already defined for service port %s, UpstreamTLSPolicy %s/%s:%d will be dropped", svcPortName.String(), upstreamTLS.Namespace, upstreamTLS.Name, p.Port)
					continue
				}

				configs[svcPortName.String()] = &UpstreamTLSConfig{
					MTLS:   cfg.MTLS,
					Secret: secret,
				}
			}
		}
	}

	return &UpstreamTLSPolicyEnricher{
		data: configs,
	}
}

// UpstreamTLSPolicyEnricher is an enricher for upstream TLS policies
type UpstreamTLSPolicyEnricher struct {
	data map[string]*UpstreamTLSConfig
}

func (e *UpstreamTLSPolicyEnricher) Enrich(svcPortName string, svcCfg *fgw.ServiceConfig) {
	if len(e.data) == 0 {
		return
	}

	if tlsCfg, exists := e.data[svcPortName]; exists {
		svcCfg.UpstreamCert = newUpstreamCert(tlsCfg)
		svcCfg.MTLS = tlsCfg.MTLS
	}
}

// ---

func NewRetryPolicyEnricher(cache cache.Cache, selector fields.Selector, targetRefToServicePortName ResolveServicePortNameFunc) ServicePolicyEnricher {
	configs := make(map[string]*gwpav1alpha1.RetryConfig)

	for _, retryPolicy := range gwutils.SortResources(gwutils.GetRetries(cache, selector)) {
		retryPolicy := retryPolicy.(*gwpav1alpha1.RetryPolicy)

		for _, p := range retryPolicy.Spec.Ports {
			if svcPortName := targetRefToServicePortName(retryPolicy, retryPolicy.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
				cfg := retry.ComputeRetryConfig(p.Config, retryPolicy.Spec.DefaultConfig)

				if cfg == nil {
					continue
				}

				if _, ok := configs[svcPortName.String()]; ok {
					log.Warn().Msgf("Policy is already defined for service port %s, RetryPolicy %s/%s:%d will be dropped", svcPortName.String(), retryPolicy.Namespace, retryPolicy.Name, p.Port)
					continue
				}

				configs[svcPortName.String()] = cfg
			}
		}
	}

	return &RetryPolicyEnricher{
		data: configs,
	}
}

// RetryPolicyEnricher is an enricher for retry policies
type RetryPolicyEnricher struct {
	data map[string]*gwpav1alpha1.RetryConfig
}

func (e *RetryPolicyEnricher) Enrich(svcPortName string, svcCfg *fgw.ServiceConfig) {
	if len(e.data) == 0 {
		return
	}

	if hcCfg, exists := e.data[svcPortName]; exists {
		svcCfg.Retry = newRetry(hcCfg)
	}
}
