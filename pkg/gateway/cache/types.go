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

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/logger"
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

type serviceContext struct {
	svcPortName fgw.ServicePortName
	//filters     []routecfg.Filter
}

type endpointContext struct {
	address string
	port    int32
}

type calculateEndpointsFunc func(svc *corev1.Service, port *int32) map[string]fgw.Endpoint

type globalPolicyAttachments struct {
	rateLimits      map[gwpkg.PolicyMatchType][]*gwpav1alpha1.RateLimitPolicy
	accessControls  map[gwpkg.PolicyMatchType][]*gwpav1alpha1.AccessControlPolicy
	faultInjections map[gwpkg.PolicyMatchType][]*gwpav1alpha1.FaultInjectionPolicy
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
