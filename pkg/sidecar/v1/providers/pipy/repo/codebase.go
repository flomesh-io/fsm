package repo

import (
	_ "embed"

	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/client"
)

//go:embed codebase/config.js
var codebaseConfigJs []byte

//go:embed codebase/connect-tcp.js
var codebaseConnectTCPJs []byte

//go:embed codebase/connect-tls.js
var codebaseConnectTLSJs []byte

//go:embed codebase/connect-upstream.js
var codebaseConnectUpstreamJs []byte

//go:embed codebase/dns-main.js
var codebaseDNSMainJs []byte

//go:embed codebase/logging.js
var codebaseLoggingJs []byte

//go:embed codebase/main.js
var codebaseMainJs []byte

//go:embed codebase/metrics.js
var codebaseMetricsJs []byte

//go:embed codebase/modules/inbound-http-default.js
var codebaseModulesInboundHTTPDefaultJs []byte

//go:embed codebase/modules/inbound-http-load-balancing.js
var codebaseModulesInboundHTTPLoadBalancingJs []byte

//go:embed codebase/modules/inbound-http-routing.js
var codebaseModulesInboundHTTPRoutingJs []byte

//go:embed codebase/modules/inbound-logging-http.js
var codebaseModulesInboundLoggingHTTPJs []byte

//go:embed codebase/modules/inbound-main.js
var codebaseModulesInboundMainJs []byte

//go:embed codebase/modules/inbound-metrics-http.js
var codebaseModulesInboundMetricsHTTPJs []byte

//go:embed codebase/modules/inbound-tcp-default.js
var codebaseModulesInboundTCPDefaultJs []byte

//go:embed codebase/modules/inbound-tcp-load-balancing.js
var codebaseModulesInboundTCPLoadBalancingJs []byte

//go:embed codebase/modules/inbound-tcp-routing.js
var codebaseModulesInboundTCPRoutingJs []byte

//go:embed codebase/modules/inbound-throttle-route.js
var codebaseModulesInboundThrottleRouteJs []byte

//go:embed codebase/modules/inbound-throttle-service.js
var codebaseModulesInboundThrottleServiceJs []byte

//go:embed codebase/modules/inbound-tls-termination.js
var codebaseModulesInboundTLSTerminationJs []byte

//go:embed codebase/modules/inbound-tracing-http.js
var codebaseModulesInboundTracingHTTPJs []byte

//go:embed codebase/modules/outbound-circuit-breaker.js
var codebaseModulesOutboundCircuitBreakerJs []byte

//go:embed codebase/modules/outbound-http-default.js
var codebaseModulesOutboundHTTPDefaultJs []byte

//go:embed codebase/modules/outbound-http-load-balancing.js
var codebaseModulesOutboundHTTPLoadBalancingJs []byte

//go:embed codebase/modules/outbound-http-routing.js
var codebaseModulesOutboundHTTPRoutingJs []byte

//go:embed codebase/modules/outbound-logging-http.js
var codebaseModulesOutboundLoggingHTTPJs []byte

//go:embed codebase/modules/outbound-main.js
var codebaseModulesOutboundMainJs []byte

//go:embed codebase/modules/outbound-metrics-http.js
var codebaseModulesOutboundMetricsHTTPJs []byte

//go:embed codebase/modules/outbound-tcp-default.js
var codebaseModulesOutboundTCPDefaultJs []byte

//go:embed codebase/modules/outbound-tcp-load-balancing.js
var codebaseModulesOutboundTCPLoadBalancingJs []byte

//go:embed codebase/modules/outbound-tcp-routing.js
var codebaseModulesOutboundTCPRoutingJs []byte

//go:embed codebase/modules/outbound-tracing-http.js
var codebaseModulesOutboundTracingHTTPJs []byte

//go:embed codebase/probes.js
var codebaseProbesJs []byte

//go:embed codebase/stats.js
var codebaseStatsJs []byte

//go:embed codebase/tracing.js
var codebaseTracingJs []byte

//go:embed codebase/utils.js
var codebaseUtilsJs []byte

//go:embed codebase/config.json
var codebaseConfigJSON []byte

var fsmCodebaseItems = []client.BatchItem{
	{Filename: "config.js", Content: codebaseConfigJs},
	{Filename: "connect-tcp.js", Content: codebaseConnectTCPJs},
	{Filename: "connect-tls.js", Content: codebaseConnectTLSJs},
	{Filename: "connect-upstream.js", Content: codebaseConnectUpstreamJs},
	{Filename: "dns-main.js", Content: codebaseDNSMainJs},
	{Filename: "logging.js", Content: codebaseLoggingJs},
	{Filename: "main.js", Content: codebaseMainJs},
	{Filename: "metrics.js", Content: codebaseMetricsJs},
	{Filename: "modules/inbound-http-default.js", Content: codebaseModulesInboundHTTPDefaultJs},
	{Filename: "modules/inbound-http-load-balancing.js", Content: codebaseModulesInboundHTTPLoadBalancingJs},
	{Filename: "modules/inbound-http-routing.js", Content: codebaseModulesInboundHTTPRoutingJs},
	{Filename: "modules/inbound-logging-http.js", Content: codebaseModulesInboundLoggingHTTPJs},
	{Filename: "modules/inbound-main.js", Content: codebaseModulesInboundMainJs},
	{Filename: "modules/inbound-metrics-http.js", Content: codebaseModulesInboundMetricsHTTPJs},
	{Filename: "modules/inbound-tcp-default.js", Content: codebaseModulesInboundTCPDefaultJs},
	{Filename: "modules/inbound-tcp-load-balancing.js", Content: codebaseModulesInboundTCPLoadBalancingJs},
	{Filename: "modules/inbound-tcp-routing.js", Content: codebaseModulesInboundTCPRoutingJs},
	{Filename: "modules/inbound-throttle-route.js", Content: codebaseModulesInboundThrottleRouteJs},
	{Filename: "modules/inbound-throttle-service.js", Content: codebaseModulesInboundThrottleServiceJs},
	{Filename: "modules/inbound-tls-termination.js", Content: codebaseModulesInboundTLSTerminationJs},
	{Filename: "modules/inbound-tracing-http.js", Content: codebaseModulesInboundTracingHTTPJs},
	{Filename: "modules/outbound-circuit-breaker.js", Content: codebaseModulesOutboundCircuitBreakerJs},
	{Filename: "modules/outbound-http-default.js", Content: codebaseModulesOutboundHTTPDefaultJs},
	{Filename: "modules/outbound-http-load-balancing.js", Content: codebaseModulesOutboundHTTPLoadBalancingJs},
	{Filename: "modules/outbound-http-routing.js", Content: codebaseModulesOutboundHTTPRoutingJs},
	{Filename: "modules/outbound-logging-http.js", Content: codebaseModulesOutboundLoggingHTTPJs},
	{Filename: "modules/outbound-main.js", Content: codebaseModulesOutboundMainJs},
	{Filename: "modules/outbound-metrics-http.js", Content: codebaseModulesOutboundMetricsHTTPJs},
	{Filename: "modules/outbound-tcp-default.js", Content: codebaseModulesOutboundTCPDefaultJs},
	{Filename: "modules/outbound-tcp-load-balancing.js", Content: codebaseModulesOutboundTCPLoadBalancingJs},
	{Filename: "modules/outbound-tcp-routing.js", Content: codebaseModulesOutboundTCPRoutingJs},
	{Filename: "modules/outbound-tracing-http.js", Content: codebaseModulesOutboundTracingHTTPJs},
	{Filename: "probes.js", Content: codebaseProbesJs},
	{Filename: "stats.js", Content: codebaseStatsJs},
	{Filename: "tracing.js", Content: codebaseTracingJs},
	{Filename: "utils.js", Content: codebaseUtilsJs},

	{Filename: fsmCodebaseConfig, Content: codebaseConfigJSON},
}
