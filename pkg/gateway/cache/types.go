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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	"github.com/flomesh-io/fsm/pkg/logger"
)

// ProcessorType is the type used to represent the type of processor
type ProcessorType string

const (
	// ServicesProcessorType is the type used to represent the services processor
	ServicesProcessorType ProcessorType = "services"

	// EndpointSlicesProcessorType is the type used to represent the endpoint slices processor
	EndpointSlicesProcessorType ProcessorType = "endpointslices"

	// EndpointsProcessorType is the type used to represent the endpoints processor
	EndpointsProcessorType ProcessorType = "endpoints"

	// ServiceImportsProcessorType is the type used to represent the service imports processor
	ServiceImportsProcessorType ProcessorType = "serviceimports"

	// SecretsProcessorType is the type used to represent the secrets processor
	SecretsProcessorType ProcessorType = "secrets"

	// GatewayClassesProcessorType is the type used to represent the gateway classes processor
	GatewayClassesProcessorType ProcessorType = "gatewayclasses"

	// GatewaysProcessorType is the type used to represent the gateways processor
	GatewaysProcessorType ProcessorType = "gateways"

	// HTTPRoutesProcessorType is the type used to represent the HTTP routes processor
	HTTPRoutesProcessorType ProcessorType = "httproutes"

	// GRPCRoutesProcessorType is the type used to represent the gRPC routes processor
	GRPCRoutesProcessorType ProcessorType = "grpcroutes"

	// TCPRoutesProcessorType is the type used to represent the TCP routes processor
	TCPRoutesProcessorType ProcessorType = "tcproutes"

	// TLSRoutesProcessorType is the type used to represent the TLS routes processor
	TLSRoutesProcessorType ProcessorType = "tlsroutes"

	// RateLimitPoliciesProcessorType is the type used to represent the rate limit policies processor
	RateLimitPoliciesProcessorType ProcessorType = "ratelimits"

	// SessionStickyPoliciesProcessorType is the type used to represent the session sticky policies processor
	SessionStickyPoliciesProcessorType ProcessorType = "sessionstickies"

	// LoadBalancerPoliciesProcessorType is the type used to represent the load balancer policies processor
	LoadBalancerPoliciesProcessorType ProcessorType = "loadbalancers"

	// CircuitBreakingPoliciesProcessorType is the type used to represent the circuit breaking policies processor
	CircuitBreakingPoliciesProcessorType ProcessorType = "circuitbreakings"

	// AccessControlPoliciesProcessorType is the type used to represent the access control policies processor
	AccessControlPoliciesProcessorType ProcessorType = "accesscontrols"

	// HealthCheckPoliciesProcessorType is the type used to represent the health check policies processor
	HealthCheckPoliciesProcessorType ProcessorType = "healthchecks"

	// FaultInjectionPoliciesProcessorType is the type used to represent the fault injection policies processor
	FaultInjectionPoliciesProcessorType ProcessorType = "faultinjections"
)

// Processor is the interface for the functionality provided by the processors
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
	svcPortName routecfg.ServicePortName
	//filters     []routecfg.Filter
}

type endpointInfo struct {
	address string
	port    int32
}

// RateLimitPolicyMatchType is the type used to represent the rate limit policy match type
type RateLimitPolicyMatchType string

const (
	// RateLimitPolicyMatchTypePort is the type used to represent the rate limit policy match type port
	RateLimitPolicyMatchTypePort RateLimitPolicyMatchType = "port"

	// RateLimitPolicyMatchTypeHostnames is the type used to represent the rate limit policy match type hostnames
	RateLimitPolicyMatchTypeHostnames RateLimitPolicyMatchType = "hostnames"

	// RateLimitPolicyMatchTypeHTTPRoute is the type used to represent the rate limit policy match type httproute
	RateLimitPolicyMatchTypeHTTPRoute RateLimitPolicyMatchType = "httproute"

	// RateLimitPolicyMatchTypeGRPCRoute is the type used to represent the rate limit policy match type grpcroute
	RateLimitPolicyMatchTypeGRPCRoute RateLimitPolicyMatchType = "grpcroute"
)

// AccessControlPolicyMatchType is the type used to represent the rate limit policy match type
type AccessControlPolicyMatchType string

const (
	// AccessControlPolicyMatchTypePort is the type used to represent the rate limit policy match type port
	AccessControlPolicyMatchTypePort AccessControlPolicyMatchType = "port"

	// AccessControlPolicyMatchTypeHostnames is the type used to represent the rate limit policy match type hostnames
	AccessControlPolicyMatchTypeHostnames AccessControlPolicyMatchType = "hostnames"

	// AccessControlPolicyMatchTypeHTTPRoute is the type used to represent the rate limit policy match type httproute
	AccessControlPolicyMatchTypeHTTPRoute AccessControlPolicyMatchType = "httproute"

	// AccessControlPolicyMatchTypeGRPCRoute is the type used to represent the rate limit policy match type grpcroute
	AccessControlPolicyMatchTypeGRPCRoute AccessControlPolicyMatchType = "grpcroute"
)

// FaultInjectionPolicyMatchType is the type used to represent the fault injection policy match type
type FaultInjectionPolicyMatchType string

const (
	// FaultInjectionPolicyMatchTypePort is the type used to represent the fault injection policy match type port
	//FaultInjectionPolicyMatchTypePort FaultInjectionPolicyMatchType = "port"

	// FaultInjectionPolicyMatchTypeHostnames is the type used to represent the fault injection policy match type hostnames
	FaultInjectionPolicyMatchTypeHostnames FaultInjectionPolicyMatchType = "hostnames"

	// FaultInjectionPolicyMatchTypeHTTPRoute is the type used to represent the fault injection policy match type httproute
	FaultInjectionPolicyMatchTypeHTTPRoute FaultInjectionPolicyMatchType = "httproute"

	// FaultInjectionPolicyMatchTypeGRPCRoute is the type used to represent the fault injection policy match type grpcroute
	FaultInjectionPolicyMatchTypeGRPCRoute FaultInjectionPolicyMatchType = "grpcroute"
)

type globalPolicyAttachments struct {
	rateLimits      map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy
	accessControls  map[AccessControlPolicyMatchType][]gwpav1alpha1.AccessControlPolicy
	faultInjections map[FaultInjectionPolicyMatchType][]gwpav1alpha1.FaultInjectionPolicy
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
)

const (
	httpCodecScript    = "http/codec.js"
	agentServiceScript = "extension/agent-service.js"
)

var (
	gatewayGVK               = schema.FromAPIVersionAndKind(gwv1beta1.GroupVersion.String(), constants.GatewayAPIGatewayKind)
	httpRouteGVK             = schema.FromAPIVersionAndKind(gwv1beta1.GroupVersion.String(), constants.GatewayAPIHTTPRouteKind)
	tlsRouteGVK              = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), constants.GatewayAPITLSRouteKind)
	tcpRouteGVK              = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), constants.GatewayAPITCPRouteKind)
	grpcRouteGVK             = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), constants.GatewayAPIGRPCRouteKind)
	secretGVK                = schema.FromAPIVersionAndKind(corev1.SchemeGroupVersion.String(), constants.KubernetesSecretKind)
	serviceGVK               = schema.FromAPIVersionAndKind(corev1.SchemeGroupVersion.String(), constants.KubernetesServiceKind)
	rateLimitPolicyGVK       = schema.FromAPIVersionAndKind(gwpav1alpha1.SchemeGroupVersion.String(), constants.RateLimitPolicyKind)
	sessionStickyPolicyGVK   = schema.FromAPIVersionAndKind(gwpav1alpha1.SchemeGroupVersion.String(), constants.SessionStickyPolicyKind)
	loadBalancerPolicyGVK    = schema.FromAPIVersionAndKind(gwpav1alpha1.SchemeGroupVersion.String(), constants.LoadBalancerPolicyKind)
	circuitBreakingPolicyGVK = schema.FromAPIVersionAndKind(gwpav1alpha1.SchemeGroupVersion.String(), constants.CircuitBreakingPolicyKind)
	accessControlPolicyGVK   = schema.FromAPIVersionAndKind(gwpav1alpha1.SchemeGroupVersion.String(), constants.AccessControlPolicyKind)
	healthCheckPolicyGVK     = schema.FromAPIVersionAndKind(gwpav1alpha1.SchemeGroupVersion.String(), constants.HealthCheckPolicyKind)
	faultInjectionPolicyGVK  = schema.FromAPIVersionAndKind(gwpav1alpha1.SchemeGroupVersion.String(), constants.FaultInjectionPolicyKind)
)
