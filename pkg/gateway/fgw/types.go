// Package routecfg contains types for the gateway route
package fgw

import (
	"fmt"
	"sort"
	"strings"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
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

func (m MatchType) Ordinal() int32 {
	return map[MatchType]int32{
		MatchTypeExact:  0,
		MatchTypePrefix: 1,
		MatchTypeRegex:  2,
	}[m]
	//return [...]string{"Exact", "Prefix", "Regex"}[m]
}

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
	EnableDebug                    bool            `json:"EnableDebug"`
	DefaultPassthroughUpstreamPort int32           `json:"DefaultPassthroughUpstreamPort"`
	StripAnyHostPort               bool            `json:"StripAnyHostPort"`
	ProxyPreserveHost              bool            `json:"ProxyPreserveHost"`
	HTTP1PerRequestLoadBalancing   bool            `json:"HTTP1PerRequestLoadBalancing"`
	HTTP2PerRequestLoadBalancing   bool            `json:"HTTP2PerRequestLoadBalancing"`
	SocketTimeout                  *int32          `json:"SocketTimeout,omitempty"`
	PidFile                        *string         `json:"PidFile,omitempty"`
	ResourceUsage                  *ResourceUsage  `json:"ResourceUsage,omitempty"`
	HealthCheckLog                 *HealthCheckLog `json:"HealthCheckLog,omitempty"`
	ProxyTag                       *ProxyTag       `json:"ProxyTag,omitempty"`
}

type ResourceUsage struct {
	ScrapeInterval int32   `json:"ScrapeInterval"`
	StorageAddress *string `json:"StorageAddress,omitempty"`
	Authorization  *string `json:"Authorization,omitempty"`
}

type HealthCheckLog struct {
	StorageAddress *string `json:"StorageAddress,omitempty"`
	Authorization  *string `json:"Authorization,omitempty"`
}

type ProxyTag struct {
	SrcHostHeader string `json:"SrcHostHeader"`
	DstHostHeader string `json:"DstHostHeader"`
}

// Listener is the listener configuration
type Listener struct {
	Protocol           gwv1.ProtocolType   `json:"Protocol"`
	Port               gwv1.PortNumber     `json:"Port"`
	Listen             gwv1.PortNumber     `json:"Listen"`
	TLS                *TLS                `json:"TLS,omitempty"`
	AccessControlLists *AccessControlLists `json:"AccessControlLists,omitempty"`
	BpsLimit           *int64              `json:"BpsLimit,omitempty"`
}

// AccessControlLists is the access control lists configuration
type AccessControlLists struct {
	Blacklist  []string `json:"Blacklist,omitempty" hash:"set"`
	Whitelist  []string `json:"Whitelist,omitempty" hash:"set"`
	EnableXFF  *bool    `json:"EnableXFF,omitempty"`
	StatusCode *int32   `json:"Status,omitempty"`
	Message    *string  `json:"Message,omitempty"`
}

// FaultInjection is the fault injection configuration
type FaultInjection struct {
	Delay *FaultInjectionDelay `json:"Delay,omitempty"`
	Abort *FaultInjectionAbort `json:"Abort,omitempty"`
}

// FaultInjectionDelay is the delay configuration for fault injection
type FaultInjectionDelay struct {
	Percent int32   `json:"Percent"`
	Fixed   *int64  `json:"Fixed,omitempty"`
	Range   *string `json:"Range,omitempty"`
	Unit    *string `json:"Unit,omitempty"`
}

// FaultInjectionAbort is the abort configuration for fault injection
type FaultInjectionAbort struct {
	Percent int32   `json:"Percent"`
	Status  *int32  `json:"Status,omitempty"`
	Message *string `json:"Message,omitempty"`
}

// TLS is the TLS configuration
type TLS struct {
	TLSModeType  gwv1.TLSModeType `json:"TLSModeType"`
	MTLS         *bool            `json:"MTLS,omitempty"`
	Certificates []Certificate    `json:"Certificates,omitempty" hash:"set"`
}

// Certificate is the certificate configuration
type Certificate struct {
	CertChain  string `json:"CertChain,omitempty"`
	PrivateKey string `json:"PrivateKey,omitempty"`
	IssuingCA  string `json:"IssuingCA,omitempty"`
}

// RouteRule is the route rule configuration
type RouteRule interface{}

// L7RouteRuleSpec is the L7 route rule configuration
type L7RouteRuleSpec interface {
	GetRateLimit() *RateLimit
	SetRateLimit(rateLimit *RateLimit)
	GetAccessControlLists() *AccessControlLists
	SetAccessControlLists(accessControlLists *AccessControlLists)
	GetFaultInjection() *FaultInjection
	SetFaultInjection(faultInjection *FaultInjection)
}

// L7RouteRule is the L7 route rule configuration
type L7RouteRule map[string]L7RouteRuleSpec

var _ RouteRule = &L7RouteRule{}

// HTTPRouteRuleSpec is the HTTP route rule configuration
type HTTPRouteRuleSpec struct {
	RouteType          L7RouteType         `json:"RouteType"`
	Matches            []HTTPTrafficMatch  `json:"Matches" hash:"set"`
	RateLimit          *RateLimit          `json:"RateLimit,omitempty"`
	AccessControlLists *AccessControlLists `json:"AccessControlLists,omitempty"`
	FaultInjection     *FaultInjection     `json:"Fault,omitempty"`
}

func (r *HTTPRouteRuleSpec) Sort() {
	if r.Matches != nil {
		matches := r.Matches
		sort.Sort(ByHTTPTrafficMatch(matches))
		r.Matches = matches
	}
}

func (r *HTTPRouteRuleSpec) GetRateLimit() *RateLimit {
	return r.RateLimit
}

func (r *HTTPRouteRuleSpec) GetAccessControlLists() *AccessControlLists {
	return r.AccessControlLists
}

func (r *HTTPRouteRuleSpec) GetFaultInjection() *FaultInjection {
	return r.FaultInjection
}

func (r *HTTPRouteRuleSpec) SetRateLimit(rateLimit *RateLimit) {
	r.RateLimit = rateLimit
}

func (r *HTTPRouteRuleSpec) SetAccessControlLists(accessControlLists *AccessControlLists) {
	r.AccessControlLists = accessControlLists
}

func (r *HTTPRouteRuleSpec) SetFaultInjection(faultInjection *FaultInjection) {
	r.FaultInjection = faultInjection
}

var _ L7RouteRuleSpec = &HTTPRouteRuleSpec{}

type ByHTTPTrafficMatch []HTTPTrafficMatch

func (by ByHTTPTrafficMatch) Len() int { return len(by) }

func (by ByHTTPTrafficMatch) Less(i, j int) bool {
	a, b := by[i], by[j]

	if a.Path != nil && b.Path == nil {
		return true
	}

	if a.Path == nil && b.Path != nil {
		return false
	}

	if a.Path == nil && b.Path == nil {
		if len(a.Methods) != len(b.Methods) {
			return len(a.Methods) > len(b.Methods)
		}

		if len(a.Headers) != len(b.Headers) {
			return len(a.Headers) > len(b.Headers)
		}

		if len(a.RequestParams) != len(b.RequestParams) {
			return len(a.RequestParams) > len(b.RequestParams)
		}
	}

	if a.Path != nil && b.Path != nil {
		if a.Path.MatchType != b.Path.MatchType {
			return a.Path.MatchType.Ordinal() < b.Path.MatchType.Ordinal()
		}

		if a.Path.Path != b.Path.Path {
			if len(a.Path.Path) == len(b.Path.Path) {
				return pathExp(a.Path.Path) > pathExp(b.Path.Path)
			}

			return len(a.Path.Path) > len(b.Path.Path)
		}

		if len(a.Methods) != len(b.Methods) {
			return len(a.Methods) > len(b.Methods)
		}

		if len(a.Headers) != len(b.Headers) {
			return len(a.Headers) > len(b.Headers)
		}

		if len(a.RequestParams) != len(b.RequestParams) {
			return len(a.RequestParams) > len(b.RequestParams)
		}
	}

	return false
}

func (by ByHTTPTrafficMatch) Swap(i, j int) { by[i], by[j] = by[j], by[i] }

func pathExp(path string) string {
	if path == "" {
		return "*"
	}

	if path == "/" {
		return "/*"
	}

	return path
}

// GRPCRouteRuleSpec is the GRPC route rule configuration
type GRPCRouteRuleSpec struct {
	RouteType          L7RouteType         `json:"RouteType"`
	Matches            []GRPCTrafficMatch  `json:"Matches" hash:"set"`
	RateLimit          *RateLimit          `json:"RateLimit,omitempty"`
	AccessControlLists *AccessControlLists `json:"AccessControlLists,omitempty"`
	FaultInjection     *FaultInjection     `json:"Fault,omitempty"`
}

func (r *GRPCRouteRuleSpec) Sort() {
	if r.Matches != nil {
		matches := r.Matches
		sort.Sort(ByGRPCTrafficMatch(matches))
		r.Matches = matches
	}
}

func (r *GRPCRouteRuleSpec) GetRateLimit() *RateLimit {
	return r.RateLimit
}

func (r *GRPCRouteRuleSpec) GetAccessControlLists() *AccessControlLists {
	return r.AccessControlLists
}

func (r *GRPCRouteRuleSpec) GetFaultInjection() *FaultInjection {
	return r.FaultInjection
}

func (r *GRPCRouteRuleSpec) SetRateLimit(rateLimit *RateLimit) {
	r.RateLimit = rateLimit
}

func (r *GRPCRouteRuleSpec) SetAccessControlLists(accessControlLists *AccessControlLists) {
	r.AccessControlLists = accessControlLists
}

func (r *GRPCRouteRuleSpec) SetFaultInjection(faultInjection *FaultInjection) {
	r.FaultInjection = faultInjection
}

var _ L7RouteRuleSpec = &GRPCRouteRuleSpec{}

type ByGRPCTrafficMatch []GRPCTrafficMatch

func (by ByGRPCTrafficMatch) Len() int { return len(by) }

func (by ByGRPCTrafficMatch) Less(i, j int) bool {
	a, b := by[i], by[j]

	if a.Method == nil && b.Method == nil {
		return len(a.Headers) > len(b.Headers)
	}

	if a.Method != nil && b.Method == nil {
		return true
	}

	if a.Method == nil && b.Method != nil {
		return false
	}

	if a.Method != nil && b.Method != nil {
		if a.Method.MatchType != b.Method.MatchType {
			return a.Method.MatchType.Ordinal() < b.Method.MatchType.Ordinal()
		}

		pathA := fmt.Sprintf("%s/%s", stringExp(a.Method.Service), stringExp(a.Method.Method))
		pathB := fmt.Sprintf("%s/%s", stringExp(b.Method.Service), stringExp(b.Method.Method))

		if len(pathA) == len(pathB) {
			switch strings.Compare(pathA, pathB) {
			case -1:
				return false
			case 1:
				return true
			case 0:
				return len(a.Headers) > len(b.Headers)
			}
		}

		return len(pathA) > len(pathB)
	}

	return false
}

func (by ByGRPCTrafficMatch) Swap(i, j int) { by[i], by[j] = by[j], by[i] }

func stringExp(s *string) string {
	if s == nil {
		return "*"
	}

	return *s
}

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
	Path               *Path                           `json:"Path,omitempty"`
	Headers            map[MatchType]map[string]string `json:"Headers,omitempty"`
	RequestParams      map[MatchType]map[string]string `json:"RequestParams,omitempty"`
	Methods            []string                        `json:"Methods,omitempty" hash:"set"`
	BackendService     map[string]BackendServiceConfig `json:"BackendService"`
	RateLimit          *RateLimit                      `json:"RateLimit,omitempty"`
	AccessControlLists *AccessControlLists             `json:"AccessControlLists,omitempty"`
	FaultInjection     *FaultInjection                 `json:"Fault,omitempty"`
	Filters            []Filter                        `json:"Filters,omitempty" hash:"set"`
}

// GRPCTrafficMatch is the GRPC traffic match configuration
type GRPCTrafficMatch struct {
	Headers            map[MatchType]map[string]string `json:"Headers,omitempty"`
	Method             *GRPCMethod                     `json:"Method,omitempty"`
	BackendService     map[string]BackendServiceConfig `json:"BackendService"`
	RateLimit          *RateLimit                      `json:"RateLimit,omitempty"`
	AccessControlLists *AccessControlLists             `json:"AccessControlLists,omitempty"`
	FaultInjection     *FaultInjection                 `json:"Fault,omitempty"`
	Filters            []Filter                        `json:"Filters,omitempty" hash:"set"`
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
	Mode                 gwpav1alpha1.RateLimitPolicyMode `json:"Mode"`
	Backlog              int32                            `json:"Backlog"`
	Requests             int32                            `json:"Requests"`
	Burst                int32                            `json:"Burst"`
	StatTimeWindow       int32                            `json:"StatTimeWindow"`
	ResponseStatusCode   int32                            `json:"ResponseStatusCode"`
	ResponseHeadersToAdd map[gwv1.HTTPHeaderName]string   `json:"ResponseHeadersToAdd,omitempty" hash:"set"`
}

// PassthroughRouteMapping is the passthrough route mapping configuration
type PassthroughRouteMapping map[string]string

// ServiceConfig is the service configuration
type ServiceConfig struct {
	Endpoints           map[string]Endpoint            `json:"Endpoints"`
	Retry               *Retry                         `json:"Retry,omitempty"`
	MTLS                *bool                          `json:"MTLS,omitempty"`
	UpstreamCert        *UpstreamCert                  `json:"UpstreamCert,omitempty"`
	StickyCookieName    *string                        `json:"StickyCookieName,omitempty"`
	StickyCookieExpires *int32                         `json:"StickyCookieExpires,omitempty"`
	LoadBalancer        *gwpav1alpha1.LoadBalancerType `json:"Algorithm,omitempty"`
	CircuitBreaking     *CircuitBreaking               `json:"CircuitBreaking,omitempty"`
	HealthCheck         *HealthCheck                   `json:"HealthCheck,omitempty"`
}

// Endpoint is the endpoint configuration
type Endpoint struct {
	Weight       int32             `json:"Weight"`
	Tags         map[string]string `json:"Tags,omitempty"`
	MTLS         bool              `json:"MTLS,omitempty"`
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
	Type               gwv1.HTTPPathModifierType `json:"Type"`
	ReplaceFullPath    *string                   `json:"ReplaceFullPath,omitempty"`
	ReplacePrefixMatch *string                   `json:"ReplacePrefixMatch,omitempty"`
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
	Type gwv1.HTTPRouteFilterType `json:"Type"`

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
	ExtensionRef *gwv1.LocalObjectReference `json:"ExtensionRef,omitempty"`
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
	Type gwv1.GRPCRouteFilterType `json:"Type"`

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

// CircuitBreaking is the circuit breaker configuration
type CircuitBreaking struct {
	MinRequestAmount        int32    `json:"MinRequestAmount"`
	StatTimeWindow          int32    `json:"StatTimeWindow"`
	SlowTimeThreshold       *float32 `json:"SlowTimeThreshold,omitempty"`
	SlowAmountThreshold     *int32   `json:"SlowAmountThreshold,omitempty"`
	SlowRatioThreshold      *float32 `json:"SlowRatioThreshold,omitempty"`
	ErrorAmountThreshold    *int32   `json:"ErrorAmountThreshold,omitempty"`
	ErrorRatioThreshold     *float32 `json:"ErrorRatioThreshold,omitempty"`
	DegradedTimeWindow      int32    `json:"DegradedTimeWindow"`
	DegradedStatusCode      int32    `json:"DegradedStatusCode"`
	DegradedResponseContent *string  `json:"DegradedResponseContent,omitempty"`
}

// HealthCheck is the health check configuration
type HealthCheck struct {
	Interval    int32              `json:"Interval"`
	MaxFails    int32              `json:"MaxFails"`
	FailTimeout *int32             `json:"FailTimeout,omitempty"`
	Path        *string            `json:"Path,omitempty"`
	Matches     []HealthCheckMatch `json:"Matches,omitempty" hash:"set"`
}

// HealthCheckMatch is the health check match configuration
type HealthCheckMatch struct {
	StatusCodes []int32                        `json:"StatusCodes,omitempty" hash:"set"`
	Body        *string                        `json:"Body,omitempty"`
	Headers     map[gwv1.HTTPHeaderName]string `json:"Headers,omitempty"`
}

// UpstreamCert is the upstream certificate configuration
type UpstreamCert Certificate

// Retry is the retry policy configuration
type Retry struct {
	RetryOn             string   `json:"RetryOn"`
	NumRetries          *int32   `json:"NumRetries,omitempty"`
	BackoffBaseInterval *float32 `json:"BackoffBaseInterval,omitempty"`
}

// Chains is the chains configuration
type Chains struct {
	HTTPRoute      []string `json:"HTTPRoute" hash:"set"`
	HTTPSRoute     []string `json:"HTTPSRoute" hash:"set"`
	TLSPassthrough []string `json:"TLSPassthrough" hash:"set"`
	TLSTerminate   []string `json:"TLSTerminate" hash:"set"`
	TCPRoute       []string `json:"TCPRoute" hash:"set"`
	UDPRoute       []string `json:"UDPRoute" hash:"set"`
}

// Features is the features configuration
type Features struct {
	Logging struct{} `json:"Logging"`
	Tracing struct{} `json:"Tracing"`
	Metrics struct{} `json:"Metrics"`
}
