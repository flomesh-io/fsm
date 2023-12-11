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
	"k8s.io/apimachinery/pkg/labels"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/logger"
)

// ResourceType is the type used to represent the type of resource
type ResourceType string

const (
	// ServicesResourceType is the type used to represent the services resource
	ServicesResourceType ResourceType = "services"

	// EndpointSlicesResourceType is the type used to represent the endpoint slices resource
	EndpointSlicesResourceType ResourceType = "endpointslices"

	// EndpointsResourceType is the type used to represent the endpoints resource
	EndpointsResourceType ResourceType = "endpoints"

	// ServiceImportsResourceType is the type used to represent the service imports resource
	ServiceImportsResourceType ResourceType = "serviceimports"

	// SecretsResourceType is the type used to represent the secrets resource
	SecretsResourceType ResourceType = "secrets"

	// GatewayClassesResourceType is the type used to represent the gateway classes resource
	GatewayClassesResourceType ResourceType = "gatewayclasses"

	// GatewaysResourceType is the type used to represent the gateways resource
	GatewaysResourceType ResourceType = "gateways"

	// HTTPRoutesResourceType is the type used to represent the HTTP routes resource
	HTTPRoutesResourceType ResourceType = "httproutes"

	// GRPCRoutesResourceType is the type used to represent the gRPC routes resource
	GRPCRoutesResourceType ResourceType = "grpcroutes"

	// TCPRoutesResourceType is the type used to represent the TCP routes resource
	TCPRoutesResourceType ResourceType = "tcproutes"

	// TLSRoutesResourceType is the type used to represent the TLS routes resource
	TLSRoutesResourceType ResourceType = "tlsroutes"

	// UDPRoutesResourceType is the type used to represent the UDP routes resource
	UDPRoutesResourceType ResourceType = "udproutes"

	// RateLimitPoliciesResourceType is the type used to represent the rate limit policies resource
	RateLimitPoliciesResourceType ResourceType = "ratelimits"

	// SessionStickyPoliciesResourceType is the type used to represent the session sticky policies resource
	SessionStickyPoliciesResourceType ResourceType = "sessionstickies"

	// LoadBalancerPoliciesResourceType is the type used to represent the load balancer policies resource
	LoadBalancerPoliciesResourceType ResourceType = "loadbalancers"

	// CircuitBreakingPoliciesResourceType is the type used to represent the circuit breaking policies resource
	CircuitBreakingPoliciesResourceType ResourceType = "circuitbreakings"

	// AccessControlPoliciesResourceType is the type used to represent the access control policies resource
	AccessControlPoliciesResourceType ResourceType = "accesscontrols"

	// HealthCheckPoliciesResourceType is the type used to represent the health check policies resource
	HealthCheckPoliciesResourceType ResourceType = "healthchecks"

	// FaultInjectionPoliciesResourceType is the type used to represent the fault injection policies resource
	FaultInjectionPoliciesResourceType ResourceType = "faultinjections"

	// UpstreamTLSPoliciesResourceType is the type used to represent the upstream tls policies resource
	UpstreamTLSPoliciesResourceType ResourceType = "upstreamtls"

	// RetryPoliciesResourceType is the type used to represent the retry policies resource
	RetryPoliciesResourceType ResourceType = "retries"

	// GatewayTLSPoliciesResourceType is the type used to represent the gateway tls policies resource
	GatewayTLSPoliciesResourceType ResourceType = "gatewaytls"
)

// Trigger is the interface for the functionality provided by the resources
type Trigger interface {
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

type GatewayAPIResource interface {
	*gwv1beta1.HTTPRoute | *gwv1alpha2.GRPCRoute | *gwv1alpha2.TLSRoute | *gwv1alpha2.TCPRoute | *gwv1alpha2.UDPRoute |
		*gwpav1alpha1.RateLimitPolicy | *gwpav1alpha1.SessionStickyPolicy | *gwpav1alpha1.LoadBalancerPolicy |
		*gwpav1alpha1.CircuitBreakingPolicy | *gwpav1alpha1.AccessControlPolicy | *gwpav1alpha1.HealthCheckPolicy |
		*gwpav1alpha1.FaultInjectionPolicy | *gwpav1alpha1.UpstreamTLSPolicy | *gwpav1alpha1.RetryPolicy | *gwpav1alpha1.GatewayTLSPolicy
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
		"http/access-control-domain.js",
		"http/access-control-route.js",
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
		"http/access-control-domain.js",
		"http/access-control-route.js",
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

var (
	selectAll = labels.Set{}.AsSelector()
)
