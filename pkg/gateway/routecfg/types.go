// Package routecfg contains types for the gateway route
package routecfg

import (
	"fmt"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"k8s.io/apimachinery/pkg/types"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	commons "github.com/flomesh-io/fsm/pkg/apis"
)

// ServicePortName is a combination of a service name, namespace, and port
type ServicePortName struct {
	types.NamespacedName
	Port *int32
}

func (spn *ServicePortName) String() string {
	return fmt.Sprintf("%s%s", spn.NamespacedName.String(), fmtPortName(spn.Port))
}

func fmtPortName(in *int32) string {
	if in == nil {
		return ""
	}
	return fmt.Sprintf(":%d", *in)
}

// MatchType is the type of match
type MatchType string

const (
	// MatchTypeExact is the exact match type
	MatchTypeExact MatchType = "Exact"

	// MatchTypePrefix is the prefix match type
	MatchTypePrefix MatchType = "Prefix"

	// MatchTypeRegex is the regex match type
	MatchTypeRegex MatchType = "Regex"
)

// L7RouteType is the type of route
type L7RouteType string

const (
	// L7RouteTypeHTTP is the HTTP route type
	L7RouteTypeHTTP L7RouteType = "HTTP"

	// L7RouteTypeGRPC is the GRPC route type
	L7RouteTypeGRPC L7RouteType = "GRPC"
)

// ConfigSpec is the configuration spec for the gateway
type ConfigSpec struct {
	Defaults    Defaults                 `json:"Configs"`
	Listeners   []Listener               `json:"Listeners" hash:"set"`
	Certificate *Certificate             `json:"Certificate,omitempty"`
	RouteRules  map[int32]RouteRule      `json:"RouteRules"`
	Services    map[string]ServiceConfig `json:"Services"`
	Chains      Chains                   `json:"Chains"`
	Features    Features                 `json:"Features"`
	Version     string                   `json:"Version" hash:"ignore"`
}

// Defaults is the default configuration
type Defaults struct {
	EnableDebug                    bool   `json:"EnableDebug"`
	DefaultPassthroughUpstreamPort uint32 `json:"DefaultPassthroughUpstreamPort"`
	StripAnyHostPort               bool   `json:"StripAnyHostPort"`
}

// Listener is the listener configuration
type Listener struct {
	Protocol           gwv1beta1.ProtocolType `json:"Protocol"`
	Port               gwv1beta1.PortNumber   `json:"Port"`
	Listen             gwv1beta1.PortNumber   `json:"Listen"`
	TLS                *TLS                   `json:"TLS,omitempty"`
	AccessControlLists *AccessControlLists    `json:"AccessControlLists,omitempty"`
	BpsLimit           *int64                 `json:"bpsLimit,omitempty"`
}

// AccessControlLists is the access control lists configuration
type AccessControlLists struct {
	Blacklist []string `json:"blacklist,omitempty"`
	Whitelist []string `json:"whitelist,omitempty"`
}

// TLS is the TLS configuration
type TLS struct {
	TLSModeType  gwv1beta1.TLSModeType `json:"TLSModeType"`
	MTLS         bool                  `json:"mTLS,omitempty"`
	Certificates []Certificate         `json:"Certificates,omitempty"`
}

// Certificate is the certificate configuration
type Certificate struct {
	CertChain  string `json:"CertChain"`
	PrivateKey string `json:"PrivateKey"`
	IssuingCA  string `json:"IssuingCA,omitempty"`
}

// RouteRule is the route rule configuration
type RouteRule interface{}

// L7RouteRuleSpec is the L7 route rule configuration
type L7RouteRuleSpec interface{}

// L7RouteRule is the L7 route rule configuration
type L7RouteRule map[string]L7RouteRuleSpec

var _ RouteRule = &L7RouteRule{}

// HTTPRouteRuleSpec is the HTTP route rule configuration
type HTTPRouteRuleSpec struct {
	RouteType L7RouteType        `json:"RouteType"`
	Matches   []HTTPTrafficMatch `json:"Matches" hash:"set"`
	RateLimit *RateLimit         `json:"RateLimit,omitempty"`
}

var _ L7RouteRuleSpec = &HTTPRouteRuleSpec{}

// GRPCRouteRuleSpec is the GRPC route rule configuration
type GRPCRouteRuleSpec struct {
	RouteType L7RouteType        `json:"RouteType"`
	Matches   []GRPCTrafficMatch `json:"Matches" hash:"set"`
}

var _ L7RouteRuleSpec = &GRPCRouteRuleSpec{}

// TLSBackendService is the TLS backend service configuration
type TLSBackendService map[string]int32

// TLSTerminateRouteRule is the TLS terminate route rule configuration
type TLSTerminateRouteRule map[string]TLSBackendService

var _ RouteRule = &TLSTerminateRouteRule{}

// TLSPassthroughRouteRule is the TLS passthrough route rule configuration
type TLSPassthroughRouteRule map[string]string

var _ RouteRule = &TLSPassthroughRouteRule{}

// TCPRouteRule is the TCP route rule configuration
type TCPRouteRule map[string]int32

var _ RouteRule = &TCPRouteRule{}

// UDPRouteRule is the UDP route rule configuration
type UDPRouteRule map[string]int32

var _ RouteRule = &UDPRouteRule{}

type BackendServiceConfig struct {
	Weight  int32    `json:"Weight"`
	Filters []Filter `json:"Filters,omitempty" hash:"set"`
}

// HTTPTrafficMatch is the HTTP traffic match configuration
type HTTPTrafficMatch struct {
	Path           *Path                           `json:"Path,omitempty"`
	Headers        map[MatchType]map[string]string `json:"Headers,omitempty"`
	RequestParams  map[MatchType]map[string]string `json:"RequestParams,omitempty"`
	Methods        []string                        `json:"Methods,omitempty" hash:"set"`
	BackendService map[string]BackendServiceConfig `json:"BackendService"`
	RateLimit      *RateLimit                      `json:"RateLimit,omitempty"`
	Filters        []Filter                        `json:"Filters,omitempty" hash:"set"`
}

// GRPCTrafficMatch is the GRPC traffic match configuration
type GRPCTrafficMatch struct {
	Headers        map[MatchType]map[string]string `json:"Headers,omitempty"`
	Method         *GRPCMethod                     `json:"Method,omitempty"`
	BackendService map[string]BackendServiceConfig `json:"BackendService"`
	Filters        []Filter                        `json:"Filters,omitempty" hash:"set"`
}

// Path is the path configuration
type Path struct {
	MatchType MatchType `json:"Type"`
	Path      string    `json:"Path"`
}

// GRPCMethod is the GRPC method configuration
type GRPCMethod struct {
	MatchType MatchType `json:"Type"`
	Service   *string   `json:"Service,omitempty"`
	Method    *string   `json:"Method,omitempty"`
}

// RateLimit is the rate limit configuration
type RateLimit struct {
	Backlog              int               `json:"Backlog"`
	Requests             int               `json:"Requests"`
	Burst                int               `json:"Burst"`
	StatTimeWindow       int               `json:"StatTimeWindow"`
	ResponseStatusCode   int               `json:"ResponseStatusCode"`
	ResponseHeadersToAdd map[string]string `json:"ResponseHeadersToAdd,omitempty" hash:"set"`
}

// PassthroughRouteMapping is the passthrough route mapping configuration
type PassthroughRouteMapping map[string]string

// ServiceConfig is the service configuration
type ServiceConfig struct {
	Endpoints          map[string]Endpoint   `json:"Endpoints"`
	ConnectionSettings *ConnectionSettings   `json:"ConnectionSettings,omitempty"`
	RetryPolicy        *RetryPolicy          `json:"RetryPolicy,omitempty"`
	MTLS               bool                  `json:"mTLS,omitempty"`
	UpstreamCert       *UpstreamCert         `json:"UpstreamCert,omitempty"`
	SessionSticky      bool                  `json:"SessionSticky,omitempty"`
	LoadBalancer       *commons.AlgoBalancer `json:"LoadBalancer,omitempty"`
}

// Endpoint is the endpoint configuration
type Endpoint struct {
	Weight       int               `json:"Weight"`
	Tags         map[string]string `json:"Tags,omitempty"`
	MTLS         bool              `json:"mTLS,omitempty"`
	UpstreamCert *UpstreamCert     `json:"UpstreamCert,omitempty"`
}

// Filter is the filter configuration
type Filter interface{}

type HTTPHeader struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type HTTPHeaderFilter struct {
	Set    []HTTPHeader `json:"Set,omitempty" hash:"set"`
	Add    []HTTPHeader `json:"Add,omitempty" hash:"set"`
	Remove []string     `json:"Remove,omitempty" hash:"set"`
}

var _ Filter = &HTTPHeaderFilter{}

type HTTPRequestMirrorFilter struct {
	BackendService string `json:"BackendService"`
}

// HTTPPathModifier defines configuration for path modifiers.
type HTTPPathModifier struct {
	Type               gwv1beta1.HTTPPathModifierType `json:"Type"`
	ReplaceFullPath    *string                        `json:"ReplaceFullPath,omitempty"`
	ReplacePrefixMatch *string                        `json:"ReplacePrefixMatch,omitempty"`
}

// HTTPURLRewriteFilter defines a filter that modifies a request during
// forwarding. At most one of these filters may be used on a Route rule.
type HTTPURLRewriteFilter struct {
	Hostname *string           `json:"Hostname,omitempty"`
	Path     *HTTPPathModifier `json:"Path,omitempty"`
}

// HTTPRequestRedirectFilter defines a filter that redirects a request. This filter
// MUST NOT be used on the same Route rule as a HTTPURLRewrite filter.
type HTTPRequestRedirectFilter struct {
	Scheme     *string           `json:"Scheme,omitempty"`
	Hostname   *string           `json:"Hostname,omitempty"`
	Path       *HTTPPathModifier `json:"Path,omitempty"`
	Port       *int32            `json:"Port,omitempty"`
	StatusCode *int              `json:"StatusCode,omitempty"`
}

// HTTPRouteFilter defines processing steps that must be completed during the
// request or response lifecycle. HTTPRouteFilters are meant as an extension
// point to express processing that may be done in Gateway implementations. Some
// examples include request or response modification, implementing
// authentication strategies, rate-limiting, and traffic shaping. API
// guarantee/conformance is defined based on the type of the filter.
type HTTPRouteFilter struct {
	// Type identifies the type of filter to apply. As with other API fields,
	// types are classified into three conformance levels:
	Type gwv1beta1.HTTPRouteFilterType `json:"Type"`

	// RequestHeaderModifier defines a schema for a filter that modifies request
	RequestHeaderModifier *HTTPHeaderFilter `json:"RequestHeaderModifier,omitempty"`

	// ResponseHeaderModifier defines a schema for a filter that modifies response
	ResponseHeaderModifier *HTTPHeaderFilter `json:"ResponseHeaderModifier,omitempty"`

	// RequestMirror defines a schema for a filter that mirrors requests.
	// Requests are sent to the specified destination, but responses from
	// that destination are ignored.
	RequestMirror *HTTPRequestMirrorFilter `json:"RequestMirror,omitempty"`

	// RequestRedirect defines a schema for a filter that responds to the
	// request with an HTTP redirection.
	RequestRedirect *HTTPRequestRedirectFilter `json:"RequestRedirect,omitempty"`

	// URLRewrite defines a schema for a filter that modifies a request during forwarding.
	URLRewrite *HTTPURLRewriteFilter `json:"UrlRewrite,omitempty"`

	// ExtensionRef is an optional, implementation-specific extension to the
	// "filter" behavior.  For example, resource "myroutefilter" in group
	// "networking.example.net"). ExtensionRef MUST NOT be used for core and
	// extended filters.
	ExtensionRef *gwv1beta1.LocalObjectReference `json:"ExtensionRef,omitempty"`
}

var _ Filter = &HTTPRouteFilter{}

// GRPCRouteFilter defines processing steps that must be completed during the
// request or response lifecycle. GRPCRouteFilters are meant as an extension
// point to express processing that may be done in Gateway implementations. Some
// examples include request or response modification, implementing
// authentication strategies, rate-limiting, and traffic shaping. API
// guarantee/conformance is defined based on the type of the filter.
type GRPCRouteFilter struct {
	// Type identifies the type of filter to apply. As with other API fields,
	// types are classified into three conformance levels:
	Type gwv1alpha2.GRPCRouteFilterType `json:"Type"`

	// RequestHeaderModifier defines a schema for a filter that modifies request
	// headers.
	RequestHeaderModifier *HTTPHeaderFilter `json:"RequestHeaderModifier,omitempty"`

	// ResponseHeaderModifier defines a schema for a filter that modifies response
	// headers.
	ResponseHeaderModifier *HTTPHeaderFilter `json:"ResponseHeaderModifier,omitempty"`

	// RequestMirror defines a schema for a filter that mirrors requests.
	// Requests are sent to the specified destination, but responses from
	// that destination are ignored.
	RequestMirror *HTTPRequestMirrorFilter `json:"RequestMirror,omitempty"`

	// ExtensionRef is an optional, implementation-specific extension to the
	// "filter" behavior.  For example, resource "myroutefilter" in group
	// "networking.example.net"). ExtensionRef MUST NOT be used for core and
	// extended filters.
	ExtensionRef *gwv1alpha2.LocalObjectReference `json:"ExtensionRef,omitempty"`
}

var _ Filter = &GRPCRouteFilter{}

// ConnectionSettings is the connection settings configuration
type ConnectionSettings struct {
	TCP  *TCPConnectionSettings  `json:"tcp,omitempty"`
	HTTP *HTTPConnectionSettings `json:"http,omitempty"`
}

// TCPConnectionSettings is the TCP connection settings configuration
type TCPConnectionSettings struct {
	MaxConnections int `json:"MaxConnections"`
}

// HTTPConnectionSettings is the HTTP connection settings configuration
type HTTPConnectionSettings struct {
	MaxRequestsPerConnection int             `json:"MaxRequestsPerConnection"`
	MaxPendingRequests       int             `json:"MaxPendingRequests"`
	CircuitBreaker           *CircuitBreaker `json:"CircuitBreaker,omitempty"`
}

// CircuitBreaker is the circuit breaker configuration
type CircuitBreaker struct {
	MinRequestAmount        int     `json:"MinRequestAmount"`
	StatTimeWindow          int     `json:"StatTimeWindow"`
	SlowTimeThreshold       float64 `json:"SlowTimeThreshold"`
	SlowAmountThreshold     int     `json:"SlowAmountThreshold"`
	SlowRatioThreshold      float64 `json:"SlowRatioThreshold"`
	ErrorAmountThreshold    int     `json:"ErrorAmountThreshold"`
	ErrorRatioThreshold     float64 `json:"ErrorRatioThreshold"`
	DegradedTimeWindow      int     `json:"DegradedTimeWindow"`
	DegradedStatusCode      int     `json:"DegradedStatusCode"`
	DegradedResponseContent string  `json:"DegradedResponseContent"`
}

// UpstreamCert is the upstream certificate configuration
type UpstreamCert Certificate

// RetryPolicy is the retry policy configuration
type RetryPolicy struct {
	RetryOn                  string `json:"RetryOn"`
	PerTryTimeout            int    `json:"PerTryTimeout"`
	NumRetries               int    `json:"NumRetries"`
	RetryBackoffBaseInterval int    `json:"RetryBackoffBaseInterval"`
}

// Chains is the chains configuration
type Chains struct {
	HTTPRoute      []string `json:"HTTPRoute"`
	HTTPSRoute     []string `json:"HTTPSRoute"`
	TLSPassthrough []string `json:"TLSPassthrough"`
	TLSTerminate   []string `json:"TLSTerminate"`
	TCPRoute       []string `json:"TCPRoute"`
}

// Features is the features configuration
type Features struct {
	Logging struct{} `json:"Logging"`
	Tracing struct{} `json:"Tracing"`
	Metrics struct{} `json:"Metrics"`
}
