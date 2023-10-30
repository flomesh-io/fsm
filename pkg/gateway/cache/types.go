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
)

const (
	// KindService is the kind used to represent the service
	KindService = "Service"

	// KindServiceImport is the kind used to represent the service import
	KindServiceImport = "ServiceImport"

	// KindSecret is the kind used to represent the secret
	KindSecret = "Secret"

	// GroupFlomeshIo is the group used to represent the flomesh.io group
	GroupFlomeshIo = "flomesh.io"

	// GroupCore is the group used to represent the core group
	GroupCore = ""
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

	// RateLimitPolicyMatchTypeRoute is the type used to represent the rate limit policy match type route
	RateLimitPolicyMatchTypeRoute RateLimitPolicyMatchType = "route"
)

//type policyAttachments struct {
//	rateLimits map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy
//}

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
		"filter/request-redirect.js",
		"filter/header-modifier.js",
		"filter/url-rewrite.js",
		"http/service.js",
		"http/metrics.js",
		"http/tracing.js",
		"http/logging.js",
		"http/circuit-breaker.js",
		"http/throttle-domain.js",
		"http/throttle-route.js",
		"http/error-page.js",
		"http/proxy-redirect.js",
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
		"filter/request-redirect.js",
		"filter/header-modifier.js",
		"filter/url-rewrite.js",
		"http/service.js",
		"http/metrics.js",
		"http/tracing.js",
		"http/logging.js",
		"http/circuit-breaker.js",
		"http/throttle-domain.js",
		"http/throttle-route.js",
		"http/error-page.js",
		"http/proxy-redirect.js",
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
