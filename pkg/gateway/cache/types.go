/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package cache contains the cache for the gateway
package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/logger"
)

// TriggerType is the type used to represent the type of trigger
type TriggerType string

const (
	// ServicesTriggerType is the type used to represent the services trigger
	ServicesTriggerType TriggerType = "services"

	// EndpointSlicesTriggerType is the type used to represent the endpoint slices trigger
	EndpointSlicesTriggerType TriggerType = "endpointslices"

	// EndpointsTriggerType is the type used to represent the endpoints trigger
	EndpointsTriggerType TriggerType = "endpoints"

	// ServiceImportsTriggerType is the type used to represent the service imports trigger
	ServiceImportsTriggerType TriggerType = "serviceimports"

	// SecretsTriggerType is the type used to represent the secrets trigger
	SecretsTriggerType TriggerType = "secrets"

	// GatewayClassesTriggerType is the type used to represent the gateway classes trigger
	GatewayClassesTriggerType TriggerType = "gatewayclasses"

	// GatewaysTriggerType is the type used to represent the gateways trigger
	GatewaysTriggerType TriggerType = "gateways"

	// HTTPRoutesTriggerType is the type used to represent the HTTP routes trigger
	HTTPRoutesTriggerType TriggerType = "httproutes"

	// GRPCRoutesTriggerType is the type used to represent the gRPC routes trigger
	GRPCRoutesTriggerType TriggerType = "grpcroutes"

	// TCPRoutesTriggerType is the type used to represent the TCP routes trigger
	TCPRoutesTriggerType TriggerType = "tcproutes"

	// TLSRoutesTriggerType is the type used to represent the TLS routes trigger
	TLSRoutesTriggerType TriggerType = "tlsroutes"

	// UDPRoutesTriggerType is the type used to represent the UDP routes trigger
	UDPRoutesTriggerType TriggerType = "udproutes"

	// RateLimitPoliciesTriggerType is the type used to represent the rate limit policies trigger
	RateLimitPoliciesTriggerType TriggerType = "ratelimits"

	// SessionStickyPoliciesTriggerType is the type used to represent the session sticky policies trigger
	SessionStickyPoliciesTriggerType TriggerType = "sessionstickies"

	// LoadBalancerPoliciesTriggerType is the type used to represent the load balancer policies trigger
	LoadBalancerPoliciesTriggerType TriggerType = "loadbalancers"

	// CircuitBreakingPoliciesTriggerType is the type used to represent the circuit breaking policies trigger
	CircuitBreakingPoliciesTriggerType TriggerType = "circuitbreakings"

	// AccessControlPoliciesTriggerType is the type used to represent the access control policies trigger
	AccessControlPoliciesTriggerType TriggerType = "accesscontrols"

	// HealthCheckPoliciesTriggerType is the type used to represent the health check policies trigger
	HealthCheckPoliciesTriggerType TriggerType = "healthchecks"

	// FaultInjectionPoliciesTriggerType is the type used to represent the fault injection policies trigger
	FaultInjectionPoliciesTriggerType TriggerType = "faultinjections"

	// UpstreamTLSPoliciesTriggerType is the type used to represent the upstream tls policies trigger
	UpstreamTLSPoliciesTriggerType TriggerType = "upstreamtls"

	// RetryPoliciesTriggerType is the type used to represent the retry policies trigger
	RetryPoliciesTriggerType TriggerType = "retries"

	// GatewayTLSPoliciesTriggerType is the type used to represent the gateway tls policies trigger
	GatewayTLSPoliciesTriggerType TriggerType = "gatewaytls"
)

// Processor is the interface for the functionality provided by the triggers
type Processor interface {
	Insert(obj interface{}, cache *GatewayCache) bool
	Delete(obj interface{}, cache *GatewayCache) bool
}

// Cache is the interface for the functionality provided by the cache
type Cache interface {
	Insert(obj interface{}) bool
	Delete(obj interface{}) bool
	BuildConfigs()
}

type serviceInfo struct {
	svcPortName fgw.ServicePortName
	//filters     []routecfg.Filter
}

type endpointInfo struct {
	address string
	port    int32
}

type globalPolicyAttachments struct {
	rateLimits      map[gwpkg.PolicyMatchType][]gwpav1alpha1.RateLimitPolicy
	accessControls  map[gwpkg.PolicyMatchType][]gwpav1alpha1.AccessControlPolicy
	faultInjections map[gwpkg.PolicyMatchType][]gwpav1alpha1.FaultInjectionPolicy
}

type routePolicies struct {
	hostnamesRateLimits      []gwpav1alpha1.RateLimitPolicy
	httpRouteRateLimits      []gwpav1alpha1.RateLimitPolicy
	grpcRouteRateLimits      []gwpav1alpha1.RateLimitPolicy
	hostnamesAccessControls  []gwpav1alpha1.AccessControlPolicy
	httpRouteAccessControls  []gwpav1alpha1.AccessControlPolicy
	grpcRouteAccessControls  []gwpav1alpha1.AccessControlPolicy
	hostnamesFaultInjections []gwpav1alpha1.FaultInjectionPolicy
	httpRouteFaultInjections []gwpav1alpha1.FaultInjectionPolicy
	grpcRouteFaultInjections []gwpav1alpha1.FaultInjectionPolicy
}

var (
	log = logger.New("fsm-gateway/cache")
)

var (
	defaultHTTPChains = []string{
		"common/access-control.js",
		"common/ratelimit.js",
		"common/consumer.js",
		"http/codec.js",
		"http/access-log.js",
		"http/auth.js",
		"http/route.js",
		"http/fault-injection.js",
		"http/service.js",
		"http/metrics.js",
		"http/tracing.js",
		"http/logging.js",
		"http/circuit-breaker.js",
		"http/throttle-domain.js",
		"http/throttle-route.js",
		"http/error-page.js",
		"http/proxy-redirect.js",
		"filter/request-redirect.js",
		"filter/header-modifier.js",
		"filter/url-rewrite.js",
		"filter/request-mirror.js",
		"http/forward.js",
		"http/default.js",
	}

	defaultHTTPSChains = []string{
		"common/access-control.js",
		"common/ratelimit.js",
		"common/tls-termination.js",
		"common/consumer.js",
		"http/codec.js",
		"http/access-log.js",
		"http/auth.js",
		"http/route.js",
		"http/fault-injection.js",
		"http/service.js",
		"http/metrics.js",
		"http/tracing.js",
		"http/logging.js",
		"http/circuit-breaker.js",
		"http/throttle-domain.js",
		"http/throttle-route.js",
		"http/error-page.js",
		"http/proxy-redirect.js",
		"filter/request-redirect.js",
		"filter/header-modifier.js",
		"filter/url-rewrite.js",
		"filter/request-mirror.js",
		"http/forward.js",
		"http/default.js",
	}

	defaultTLSPassthroughChains = []string{
		"common/access-control.js",
		"common/ratelimit.js",
		"tls/passthrough.js",
		"common/consumer.js",
	}

	defaultTLSTerminateChains = []string{
		"common/access-control.js",
		"common/ratelimit.js",
		"common/tls-termination.js",
		"common/consumer.js",
		"tls/forward.js",
	}

	defaultTCPChains = []string{
		"common/access-control.js",
		"common/ratelimit.js",
		"tcp/forward.js",
	}

	defaultUDPChains = []string{
		"common/access-control.js",
		"udp/forward.js",
	}
)

const (
	httpCodecScript    = "http/codec.js"
	agentServiceScript = "extension/agent-service.js"
	proxyTagScript     = "extension/proxy-tag.js"
)
