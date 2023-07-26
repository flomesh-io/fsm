package route

import (
	"fmt"
	commons "github.com/flomesh-io/fsm/pkg/apis"
	"k8s.io/apimachinery/pkg/types"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

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

type MatchType string

const (
	MatchTypeExact  MatchType = "Exact"
	MatchTypePrefix MatchType = "Prefix"
	MatchTypeRegex  MatchType = "Regex"
)

type RouteType string

const (
	RouteTypeHTTP RouteType = "HTTP"
	RouteTypeGRPC RouteType = "GRPC"
)

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

type Defaults struct {
	EnableDebug                    bool   `json:"EnableDebug"`
	DefaultPassthroughUpstreamPort uint32 `json:"DefaultPassthroughUpstreamPort"`
}

type Listener struct {
	Protocol           gwv1beta1.ProtocolType `json:"Protocol"`
	Port               gwv1beta1.PortNumber   `json:"Port"`
	Listen             gwv1beta1.PortNumber   `json:"Listen"`
	TLS                *TLS                   `json:"TLS,omitempty"`
	AccessControlLists *AccessControlLists    `json:"AccessControlLists,omitempty"`
	BpsLimit           *int64                 `json:"bpsLimit,omitempty"`
}

type AccessControlLists struct {
	Blacklist []string `json:"blacklist,omitempty"`
	Whitelist []string `json:"whitelist,omitempty"`
}

type TLS struct {
	TLSModeType  gwv1beta1.TLSModeType `json:"TLSModeType"`
	MTLS         bool                  `json:"mTLS,omitempty"`
	Certificates []Certificate         `json:"Certificates,omitempty"`
}

type Certificate struct {
	CertChain  string `json:"CertChain"`
	PrivateKey string `json:"PrivateKey"`
	IssuingCA  string `json:"IssuingCA,omitempty"`
}

type RouteRule interface{}
type L7RouteRuleSpec interface{}

type L7RouteRule map[string]L7RouteRuleSpec

var _ RouteRule = &L7RouteRule{}

type HTTPRouteRuleSpec struct {
	RouteType RouteType          `json:"RouteType"`
	Matches   []HTTPTrafficMatch `json:"Matches" hash:"set"`
	RateLimit *RateLimit         `json:"RateLimit,omitempty"`
}

var _ L7RouteRuleSpec = &HTTPRouteRuleSpec{}

type GRPCRouteRuleSpec struct {
	RouteType RouteType          `json:"RouteType"`
	Matches   []GRPCTrafficMatch `json:"Matches" hash:"set"`
}

var _ L7RouteRuleSpec = &GRPCRouteRuleSpec{}

type TLSBackendService map[string]int32
type TLSTerminateRouteRule map[string]TLSBackendService

var _ RouteRule = &TLSTerminateRouteRule{}

type TLSPassthroughRouteRule map[string]string

var _ RouteRule = &TLSPassthroughRouteRule{}

type TCPRouteRule map[string]int32

var _ RouteRule = &TCPRouteRule{}

type UDPRouteRule map[string]int32

var _ RouteRule = &UDPRouteRule{}

type HTTPTrafficMatch struct {
	Path           *Path                           `json:"Path,omitempty"`
	Headers        map[MatchType]map[string]string `json:"Headers,omitempty"`
	RequestParams  map[MatchType]map[string]string `json:"RequestParams,omitempty"`
	Methods        []string                        `json:"Methods,omitempty" hash:"set"`
	BackendService map[string]int32                `json:"BackendService"`
	RateLimit      *RateLimit                      `json:"RateLimit,omitempty"`
}

type GRPCTrafficMatch struct {
	Headers        map[MatchType]map[string]string `json:"Headers,omitempty"`
	Method         *GRPCMethod                     `json:"Method,omitempty"`
	BackendService map[string]int32                `json:"BackendService"`
}

type Path struct {
	MatchType MatchType `json:"Type"`
	Path      string    `json:"Path"`
}

type GRPCMethod struct {
	MatchType MatchType `json:"Type"`
	Service   *string   `json:"Service,omitempty"`
	Method    *string   `json:"Method,omitempty"`
}

type RateLimit struct {
	Backlog              int               `json:"Backlog"`
	Requests             int               `json:"Requests"`
	Burst                int               `json:"Burst"`
	StatTimeWindow       int               `json:"StatTimeWindow"`
	ResponseStatusCode   int               `json:"ResponseStatusCode"`
	ResponseHeadersToAdd map[string]string `json:"ResponseHeadersToAdd,omitempty" hash:"set"`
}

type PassthroughRouteMapping map[string]string

type ServiceConfig struct {
	Endpoints          map[string]Endpoint   `json:"Endpoints"`
	Filters            []Filter              `json:"Filters,omitempty" hash:"set"`
	ConnectionSettings *ConnectionSettings   `json:"ConnectionSettings,omitempty"`
	RetryPolicy        *RetryPolicy          `json:"RetryPolicy,omitempty"`
	MTLS               bool                  `json:"mTLS,omitempty"`
	UpstreamCert       *UpstreamCert         `json:"UpstreamCert,omitempty"`
	SessionSticky      bool                  `json:"SessionSticky,omitempty"`
	LoadBalancer       *commons.AlgoBalancer `json:"LoadBalancer,omitempty"`
}

type Endpoint struct {
	Weight       int               `json:"Weight"`
	Tags         map[string]string `json:"Tags,omitempty"`
	MTLS         bool              `json:"mTLS,omitempty"`
	UpstreamCert *UpstreamCert     `json:"UpstreamCert,omitempty"`
}

type Filter interface{}

var _ Filter = &gwv1beta1.HTTPRouteFilter{}
var _ Filter = &gwv1alpha2.GRPCRouteFilter{}

type ConnectionSettings struct {
	TCP  *TCPConnectionSettings  `json:"tcp,omitempty"`
	HTTP *HTTPConnectionSettings `json:"http,omitempty"`
}

type TCPConnectionSettings struct {
	MaxConnections int `json:"MaxConnections"`
}

type HTTPConnectionSettings struct {
	MaxRequestsPerConnection int             `json:"MaxRequestsPerConnection"`
	MaxPendingRequests       int             `json:"MaxPendingRequests"`
	CircuitBreaker           *CircuitBreaker `json:"CircuitBreaker,omitempty"`
}

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

type UpstreamCert Certificate

type RetryPolicy struct {
	RetryOn                  string `json:"RetryOn"`
	PerTryTimeout            int    `json:"PerTryTimeout"`
	NumRetries               int    `json:"NumRetries"`
	RetryBackoffBaseInterval int    `json:"RetryBackoffBaseInterval"`
}

type Chains struct {
	HTTPRoute      []string `json:"HTTPRoute"`
	HTTPSRoute     []string `json:"HTTPSRoute"`
	TLSPassthrough []string `json:"TLSPassthrough"`
	TLSTerminate   []string `json:"TLSTerminate"`
	TCPRoute       []string `json:"TCPRoute"`
}

type Features struct {
	Logging struct{} `json:"Logging"`
	Tracing struct{} `json:"Tracing"`
	Metrics struct{} `json:"Metrics"`
}
