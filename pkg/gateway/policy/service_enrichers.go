package policy

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
)

// ServicePolicyEnricher is an interface for enriching service level policies
type ServicePolicyEnricher interface {
	// Enrich enriches the service config with the service level policy based on the service port name
	Enrich(svcPortName string, svcCfg *routecfg.ServiceConfig)
}

// ---

// SessionStickyPolicyEnricher is an enricher for session sticky policies
type SessionStickyPolicyEnricher struct {
	Data map[string]*gwpav1alpha1.SessionStickyConfig
}

func (e *SessionStickyPolicyEnricher) Enrich(svcPortName string, svcCfg *routecfg.ServiceConfig) {
	if len(e.Data) == 0 {
		return
	}

	if ssCfg, exists := e.Data[svcPortName]; exists {
		svcCfg.StickyCookieName = ssCfg.CookieName
		svcCfg.StickyCookieExpires = ssCfg.Expires
	}
}

// ---

// LoadBalancerPolicyEnricher is an enricher for load balancer policies
type LoadBalancerPolicyEnricher struct {
	Data map[string]*gwpav1alpha1.LoadBalancerType
}

func (e *LoadBalancerPolicyEnricher) Enrich(svcPortName string, svcCfg *routecfg.ServiceConfig) {
	if len(e.Data) == 0 {
		return
	}

	if lbType, exists := e.Data[svcPortName]; exists {
		svcCfg.LoadBalancer = lbType
	}
}

// ---

// CircuitBreakingPolicyEnricher is an enricher for circuit breaking policies
type CircuitBreakingPolicyEnricher struct {
	Data map[string]*gwpav1alpha1.CircuitBreakingConfig
}

func (e *CircuitBreakingPolicyEnricher) Enrich(svcPortName string, svcCfg *routecfg.ServiceConfig) {
	if len(e.Data) == 0 {
		return
	}

	if cbCfg, exists := e.Data[svcPortName]; exists {
		svcCfg.CircuitBreaking = newCircuitBreaking(cbCfg)
	}
}

// ---

// HealthCheckPolicyEnricher is an enricher for health check policies
type HealthCheckPolicyEnricher struct {
	Data map[string]*gwpav1alpha1.HealthCheckConfig
}

func (e *HealthCheckPolicyEnricher) Enrich(svcPortName string, svcCfg *routecfg.ServiceConfig) {
	if len(e.Data) == 0 {
		return
	}

	if hcCfg, exists := e.Data[svcPortName]; exists {
		svcCfg.HealthCheck = newHealthCheck(hcCfg)
	}
}

// ---

// UpstreamTLSPolicyEnricher is an enricher for upstream TLS policies
type UpstreamTLSPolicyEnricher struct {
	Data map[string]*UpstreamTLSConfig
}

func (e *UpstreamTLSPolicyEnricher) Enrich(svcPortName string, svcCfg *routecfg.ServiceConfig) {
	if len(e.Data) == 0 {
		return
	}

	if tlsCfg, exists := e.Data[svcPortName]; exists {
		svcCfg.UpstreamCert = newUpstreamCert(tlsCfg)
		svcCfg.MTLS = tlsCfg.MTLS
	}
}
