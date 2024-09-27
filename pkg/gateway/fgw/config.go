package fgw

import (
	"encoding/json"
	"fmt"
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"k8s.io/utils/ptr"

	"github.com/google/go-cmp/cmp"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// ---

type ConfigSpec struct {
	Resources []Resource                                                       `json:"resources" hash:"set"`
	Secrets   map[string]string                                                `json:"secrets"`
	Filters   map[extv1alpha1.FilterProtocol]map[extv1alpha1.FilterType]string `json:"filters"`
	Version   string                                                           `json:"version" hash:"ignore"`
}

func (c *ConfigSpec) GetVersion() string {
	return c.Version
}

func (c *ConfigSpec) GetResources() []Resource {
	return c.Resources
}

func (c *ConfigSpec) GetSecrets() map[string]string {
	return c.Secrets
}

func (c *ConfigSpec) GetFilters() map[extv1alpha1.FilterProtocol]map[extv1alpha1.FilterType]string {
	return c.Filters
}

// ---

type ObjectMeta struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

// ---

type CommonRouteSpec struct {
	ParentRefs []gwv1.ParentReference `json:"parentRefs,omitempty" hash:"set"`
}

// ---

type CommonResource struct {
	Kind       string     `json:"kind"`
	ObjectMeta ObjectMeta `json:"metadata"`
}

func (r *CommonResource) GetKind() string {
	return r.Kind
}

func (r *CommonResource) GetNamespace() string {
	return r.ObjectMeta.Namespace
}

func (r *CommonResource) GetName() string {
	return r.ObjectMeta.Name
}

// ---

type Gateway struct {
	CommonResource `json:",inline"`
	Spec           GatewaySpec `json:"spec"`
}

type GatewaySpec struct {
	GatewayClassName gwv1.ObjectName       `json:"gatewayClassName"`
	Listeners        []Listener            `json:"listeners,omitempty" copier:"-" hash:"set"`
	Addresses        []gwv1.GatewayAddress `json:"addresses,omitempty"`
}

type Listener struct {
	Name     gwv1.SectionName  `json:"name"`
	Hostname *gwv1.Hostname    `json:"hostname,omitempty"`
	Port     gwv1.PortNumber   `json:"port"`
	Protocol gwv1.ProtocolType `json:"protocol"`
	TLS      *GatewayTLSConfig `json:"tls,omitempty" copier:"-"`
	Filters  []ListenerFilter  `json:"routeFilters,omitempty" hash:"set" copier:"-"`
}

type ListenerFilter struct {
	Type            extv1alpha1.FilterType `json:"type"`
	ExtensionConfig map[string]interface{} `json:"-"`
	Key             string                 `json:"key,omitempty"`
}

func (f ListenerFilter) MarshalJSON() ([]byte, error) {
	type LF ListenerFilter
	b, _ := json.Marshal(LF(f))

	var m map[string]json.RawMessage
	_ = json.Unmarshal(b, &m)

	for k, v := range f.ExtensionConfig {
		b, _ = json.Marshal(v)
		m[k] = b
	}

	return json.Marshal(m)
}

type GatewayTLSConfig struct {
	Mode               *gwv1.TLSModeType                           `json:"mode,omitempty"`
	Certificates       []map[string]string                         `json:"certificates,omitempty" copier:"-" hash:"set"`
	FrontendValidation *FrontendTLSValidation                      `json:"frontendValidation,omitempty" copier:"-"`
	Options            map[gwv1.AnnotationKey]gwv1.AnnotationValue `json:"options,omitempty"`
}

type FrontendTLSValidation struct {
	CACertificates []map[string]string `json:"caCertificates,omitempty" copier:"-" hash:"set"`
}

// ---

type HTTPRoute struct {
	CommonResource `json:",inline"`
	Spec           HTTPRouteSpec `json:"spec"`
}

type HTTPRouteSpec struct {
	CommonRouteSpec `json:",inline"`
	Hostnames       []gwv1.Hostname `json:"hostnames,omitempty" hash:"set"`
	Rules           []HTTPRouteRule `json:"rules,omitempty" copier:"-" hash:"set"`
}
type HTTPRouteRule struct {
	Matches            []gwv1.HTTPRouteMatch    `json:"matches,omitempty" hash:"set"`
	Filters            []HTTPRouteFilter        `json:"filters,omitempty" hash:"set"`
	BackendRefs        []HTTPBackendRef         `json:"backendRefs,omitempty" copier:"-" hash:"set"`
	Timeouts           *gwv1.HTTPRouteTimeouts  `json:"timeouts,omitempty"`
	SessionPersistence *gwv1.SessionPersistence `json:"sessionPersistence,omitempty"`
}

// ---

type GRPCRoute struct {
	CommonResource `json:",inline"`
	Spec           GRPCRouteSpec `json:"spec,omitempty"`
}

type GRPCRouteSpec struct {
	CommonRouteSpec `json:",inline"`
	Hostnames       []gwv1.Hostname `json:"hostnames,omitempty" hash:"set"`
	Rules           []GRPCRouteRule `json:"rules,omitempty" copier:"-" hash:"set"`
}

type GRPCRouteRule struct {
	Matches            []gwv1.GRPCRouteMatch    `json:"matches,omitempty" hash:"set"`
	Filters            []GRPCRouteFilter        `json:"filters,omitempty" hash:"set"`
	BackendRefs        []GRPCBackendRef         `json:"backendRefs,omitempty" copier:"-" hash:"set"`
	SessionPersistence *gwv1.SessionPersistence `json:"sessionPersistence,omitempty"`
}

// ---

type TCPRoute struct {
	CommonResource `json:",inline"`
	Spec           TCPRouteSpec `json:"spec"`
}

// TCPRouteSpec defines the desired state of TCPRoute
type TCPRouteSpec struct {
	CommonRouteSpec `json:",inline"`
	Rules           []TCPRouteRule `json:"rules" copier:"-" hash:"set"`
}

type TCPRouteRule struct {
	BackendRefs []BackendRef `json:"backendRefs,omitempty" copier:"-" hash:"set"`
}

// ---

type TLSRoute struct {
	CommonResource `json:",inline"`
	Spec           TLSRouteSpec `json:"spec"`
}

// TLSRouteSpec defines the desired state of a TLSRoute resource.
type TLSRouteSpec struct {
	CommonRouteSpec `json:",inline"`
	Hostnames       []gwv1alpha2.Hostname `json:"hostnames,omitempty" hash:"set"`
	Rules           []TLSRouteRule        `json:"rules" copier:"-" hash:"set"`
}

type TLSRouteRule struct {
	BackendRefs []BackendRef `json:"backendRefs,omitempty" hash:"set"`
}

// ---

type UDPRoute struct {
	CommonResource `json:",inline"`
	Spec           UDPRouteSpec `json:"spec"`
}

type UDPRouteSpec struct {
	CommonRouteSpec `json:",inline"`
	Rules           []UDPRouteRule `json:"rules" copier:"-" hash:"set"`
}

type UDPRouteRule struct {
	BackendRefs []BackendRef `json:"backendRefs,omitempty" copier:"-" hash:"set"`
}

// ---

type HTTPBackendRef struct {
	Kind    string            `json:"kind"`
	Name    string            `json:"name"`
	Weight  int32             `json:"weight,omitempty"`
	Filters []HTTPRouteFilter `json:"filters,omitempty" hash:"set"`
}

func NewHTTPBackendRef(name string, weight int32) HTTPBackendRef {
	return HTTPBackendRef{
		Kind:   "Backend",
		Name:   name,
		Weight: weight,
	}
}

// ---

type HTTPRouteFilter struct {
	Type                   gwv1.HTTPRouteFilterType        `json:"type"`
	RequestHeaderModifier  *gwv1.HTTPHeaderFilter          `json:"requestHeaderModifier,omitempty"`
	ResponseHeaderModifier *gwv1.HTTPHeaderFilter          `json:"responseHeaderModifier,omitempty"`
	RequestMirror          *HTTPRequestMirrorFilter        `json:"requestMirror,omitempty"`
	RequestRedirect        *gwv1.HTTPRequestRedirectFilter `json:"requestRedirect,omitempty"`
	URLRewrite             *gwv1.HTTPURLRewriteFilter      `json:"urlRewrite,omitempty"`
	ExtensionConfig        map[string]interface{}          `json:"-"`
	Key                    string                          `json:"key,omitempty"`
}

func (f HTTPRouteFilter) MarshalJSON() ([]byte, error) {
	type HRF HTTPRouteFilter
	b, _ := json.Marshal(HRF(f))

	var m map[string]json.RawMessage
	_ = json.Unmarshal(b, &m)

	for k, v := range f.ExtensionConfig {
		b, _ = json.Marshal(v)
		m[k] = b
	}

	return json.Marshal(m)
}

// ---

type HTTPRequestMirrorFilter struct {
	BackendRef BackendRef `json:"backendRef"`
}

// ---

type GRPCBackendRef struct {
	Kind    string            `json:"kind"`
	Name    string            `json:"name"`
	Weight  int32             `json:"weight,omitempty"`
	Filters []GRPCRouteFilter `json:"filters,omitempty" hash:"set"`
}

func NewGRPCBackendRef(name string, weight int32) GRPCBackendRef {
	return GRPCBackendRef{
		Kind:   "Backend",
		Name:   name,
		Weight: weight,
	}
}

// ---

type GRPCRouteFilter struct {
	Type                   gwv1.GRPCRouteFilterType `json:"type"`
	RequestHeaderModifier  *gwv1.HTTPHeaderFilter   `json:"requestHeaderModifier,omitempty"`
	ResponseHeaderModifier *gwv1.HTTPHeaderFilter   `json:"responseHeaderModifier,omitempty"`
	RequestMirror          *HTTPRequestMirrorFilter `json:"requestMirror,omitempty"`
	ExtensionConfig        map[string]interface{}   `json:"-"`
	Key                    string                   `json:"key,omitempty"`
}

func (f GRPCRouteFilter) MarshalJSON() ([]byte, error) {
	type GRF GRPCRouteFilter
	b, _ := json.Marshal(GRF(f))

	var m map[string]json.RawMessage
	_ = json.Unmarshal(b, &m)

	for k, v := range f.ExtensionConfig {
		b, _ = json.Marshal(v)
		m[k] = b
	}

	return json.Marshal(m)
}

// ---

type BackendRef struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Weight *int32 `json:"weight,omitempty"`
}

func NewBackendRef(name string) BackendRef {
	return BackendRef{
		Kind: "Backend",
		Name: name,
	}
}

func NewBackendRefWithWeight(name string, weight int32) BackendRef {
	return BackendRef{
		Kind:   "Backend",
		Name:   name,
		Weight: ptr.To(weight),
	}
}

type Backend struct {
	CommonResource `json:",inline"`
	Spec           BackendSpec `json:"spec"`
	Port           int32       `json:"-"` // store the port for the backend temporarily
}

func NewBackend(svcPortName string, targets []BackendTarget) *Backend {
	return &Backend{
		CommonResource: CommonResource{
			Kind: "Backend",
			ObjectMeta: ObjectMeta{
				Name: svcPortName,
			},
		},
		Spec: BackendSpec{
			Targets: targets,
		},
	}
}

type BackendSpec struct {
	Targets []BackendTarget `json:"targets,omitempty" hash:"set"`
}

type BackendTarget struct {
	Address string            `json:"address"`
	Port    *int32            `json:"port"`
	Weight  int32             `json:"weight,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
}

// ---

// ServicePortName is a combination of a service name, namespace, and port
type ServicePortName struct {
	types.NamespacedName
	SectionName string
	Port        *int32
}

func (spn *ServicePortName) String() string {
	return fmt.Sprintf("%s-%s%s", spn.Namespace, spn.Name, fmtPortName(spn.Port))
}

func fmtPortName(in *int32) string {
	if in == nil {
		return ""
	}
	return fmt.Sprintf("-%d", *in)
}

// ---

type BackendTLSPolicy struct {
	CommonResource `json:",inline"`
	Spec           BackendTLSPolicySpec `json:"spec"`
}

type BackendTLSPolicySpec struct {
	TargetRefs []BackendRef               `json:"targetRefs" copier:"-" hash:"set"`
	Validation BackendTLSPolicyValidation `json:"validation"`
}

type BackendTLSPolicyValidation struct {
	CACertificates          []map[string]string                     `json:"caCertificates,omitempty" copier:"-" hash:"set"`
	WellKnownCACertificates *gwv1alpha3.WellKnownCACertificatesType `json:"wellKnownCACertificates,omitempty"`
	Hostname                gwv1.PreciseHostname                    `json:"hostname"`
}

func (p *BackendTLSPolicy) AddTargetRef(ref BackendRef) {
	if len(p.Spec.TargetRefs) > 0 {
		exists := false
		for _, targetRef := range p.Spec.TargetRefs {
			if cmp.Equal(targetRef, ref) {
				exists = true
				break
			}
		}

		if !exists {
			p.Spec.TargetRefs = append(p.Spec.TargetRefs, ref)
		}
	} else {
		p.Spec.TargetRefs = []BackendRef{ref}
	}
}

// ---

type BackendLBPolicy struct {
	CommonResource `json:",inline"`
	Spec           BackendLBPolicySpec `json:"spec"`
}

type BackendLBPolicySpec struct {
	TargetRefs         []BackendRef                        `json:"targetRefs" copier:"-" hash:"set"`
	SessionPersistence *gwv1.SessionPersistence            `json:"sessionPersistence,omitempty"`
	Algorithm          *gwpav1alpha2.LoadBalancerAlgorithm `json:"algorithm,omitempty"`
}

func (p *BackendLBPolicy) AddTargetRef(ref BackendRef) {
	if len(p.Spec.TargetRefs) > 0 {
		exists := false
		for _, targetRef := range p.Spec.TargetRefs {
			if cmp.Equal(targetRef, ref) {
				exists = true
				break
			}
		}

		if !exists {
			p.Spec.TargetRefs = append(p.Spec.TargetRefs, ref)
		}
	} else {
		p.Spec.TargetRefs = []BackendRef{ref}
	}
}

// ---

type RetryPolicy struct {
	CommonResource `json:",inline"`
	Spec           RetryPolicySpec `json:"spec"`
}

type RetryPolicySpec struct {
	TargetRefs   []BackendRef `json:"targetRefs" copier:"-" hash:"set"`
	Ports        []PortRetry  `json:"ports,omitempty" hash:"set"`
	DefaultRetry *RetryConfig `json:"retry,omitempty"`
}

type PortRetry struct {
	Port  gwv1.PortNumber `json:"port"`
	Retry *RetryConfig    `json:"retry,omitempty"`
}

type RetryConfig struct {
	RetryOn                      []string `json:"retryOn,omitempty"`
	NumRetries                   *int32   `json:"numRetries,omitempty"`
	BackoffBaseIntervalInSeconds *float64 `json:"backoffBaseInterval,omitempty"`
}

func (p *RetryPolicy) AddTargetRef(ref BackendRef) {
	if len(p.Spec.TargetRefs) > 0 {
		exists := false
		for _, targetRef := range p.Spec.TargetRefs {
			if cmp.Equal(targetRef, ref) {
				exists = true
				break
			}
		}

		if !exists {
			p.Spec.TargetRefs = append(p.Spec.TargetRefs, ref)
		}
	} else {
		p.Spec.TargetRefs = []BackendRef{ref}
	}
}

func (p *RetryPolicy) AddPort(port gwpav1alpha2.PortRetry) {
	if len(p.Spec.Ports) > 0 {
		exists := false
		for _, p := range p.Spec.Ports {
			if p.Port == port.Port {
				exists = true
				break
			}
		}

		if !exists {
			p.Spec.Ports = append(p.Spec.Ports, PortRetry{Port: port.Port, Retry: newRetryConfig(port.Retry)})
		}
	} else {
		p.Spec.Ports = []PortRetry{{Port: port.Port, Retry: newRetryConfig(port.Retry)}}
	}
}

func newRetryConfig(config *gwpav1alpha2.RetryConfig) *RetryConfig {
	if config == nil {
		return nil
	}

	c := &RetryConfig{
		RetryOn:    config.RetryOn,
		NumRetries: config.NumRetries,
	}
	c.BackoffBaseInterval(config.BackoffBaseInterval)

	return c
}

func (c *RetryConfig) BackoffBaseInterval(backoffBaseInterval *metav1.Duration) {
	if backoffBaseInterval != nil {
		c.BackoffBaseIntervalInSeconds = ptr.To(math.Ceil(backoffBaseInterval.Seconds()*1000) / 1000)
	}
}

// ---

type HealthCheckPolicy struct {
	CommonResource `json:",inline"`
	Spec           HealthCheckPolicySpec `json:"spec"`
}

type HealthCheckPolicySpec struct {
	TargetRefs         []BackendRef       `json:"targetRefs" copier:"-" hash:"set"`
	Ports              []PortHealthCheck  `json:"ports,omitempty" copier:"-" hash:"set"`
	DefaultHealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

type PortHealthCheck struct {
	Port        gwv1.PortNumber    `json:"port"`
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

type HealthCheckConfig struct {
	IntervalInSeconds    float64                         `json:"interval"`
	MaxFails             int32                           `json:"maxFails"`
	FailTimeoutInSeconds *float64                        `json:"failTimeout,omitempty"`
	Path                 *string                         `json:"path,omitempty"`
	Matches              []gwpav1alpha2.HealthCheckMatch `json:"matches,omitempty"`
}

func (p *HealthCheckPolicy) AddTargetRef(ref BackendRef) {
	if len(p.Spec.TargetRefs) > 0 {
		exists := false
		for _, targetRef := range p.Spec.TargetRefs {
			if cmp.Equal(targetRef, ref) {
				exists = true
				break
			}
		}

		if !exists {
			p.Spec.TargetRefs = append(p.Spec.TargetRefs, ref)
		}
	} else {
		p.Spec.TargetRefs = []BackendRef{ref}
	}
}

func (p *HealthCheckPolicy) AddPort(port gwpav1alpha2.PortHealthCheck) {
	if len(p.Spec.Ports) > 0 {
		exists := false
		for _, p := range p.Spec.Ports {
			if p.Port == port.Port {
				exists = true
				break
			}
		}

		if !exists {
			p.Spec.Ports = append(p.Spec.Ports, PortHealthCheck{Port: port.Port, HealthCheck: newHealthCheckConfig(port.HealthCheck)})
		}
	} else {
		p.Spec.Ports = []PortHealthCheck{{Port: port.Port, HealthCheck: newHealthCheckConfig(port.HealthCheck)}}
	}
}

func newHealthCheckConfig(config *gwpav1alpha2.HealthCheckConfig) *HealthCheckConfig {
	if config == nil {
		return nil
	}

	c := &HealthCheckConfig{
		MaxFails: config.MaxFails,
		Path:     config.Path,
		Matches:  config.Matches,
	}
	c.Interval(config.Interval)
	c.FailTimeout(config.FailTimeout)

	return c
}

func (c *HealthCheckConfig) Interval(interval metav1.Duration) {
	c.IntervalInSeconds = math.Ceil(interval.Seconds()*1000) / 1000
}

func (c *HealthCheckConfig) FailTimeout(failTimeout *metav1.Duration) {
	if failTimeout != nil {
		c.FailTimeoutInSeconds = ptr.To(math.Ceil(failTimeout.Seconds()*1000) / 1000)
	}
}

// ---

type CircuitBreakerSpec struct {
	LatencyThresholdInMilliseconds *int64                              `json:"latencyThreshold,omitempty"`
	ErrorCountThreshold            *int32                              `json:"errorCountThreshold,omitempty"`
	ErrorRatioThreshold            *float32                            `json:"errorRatioThreshold,omitempty"`
	ConcurrencyThreshold           *int32                              `json:"concurrencyThreshold,omitempty"`
	CheckIntervalInMilliseconds    *int64                              `json:"checkInterval,omitempty"`
	BreakIntervalInMilliseconds    *int64                              `json:"breakInterval,omitempty"`
	CircuitBreakerResponse         *extv1alpha1.CircuitBreakerResponse `json:"response,omitempty"`
}

func (c *CircuitBreakerSpec) LatencyThreshold(latencyThreshold *metav1.Duration) {
	if latencyThreshold != nil {
		c.LatencyThresholdInMilliseconds = ptr.To(latencyThreshold.Milliseconds())
	}
}

func (c *CircuitBreakerSpec) CheckInterval(checkInterval *metav1.Duration) {
	if checkInterval != nil {
		c.CheckIntervalInMilliseconds = ptr.To(checkInterval.Milliseconds())
	}
}

func (c *CircuitBreakerSpec) BreakInterval(breakInterval *metav1.Duration) {
	if breakInterval != nil {
		c.BreakIntervalInMilliseconds = ptr.To(breakInterval.Milliseconds())
	}
}

// ---

type FaultInjectionSpec struct {
	Delay *FaultInjectionDelay             `json:"delay,omitempty"`
	Abort *extv1alpha1.FaultInjectionAbort `json:"abort,omitempty"`
}

type FaultInjectionDelay struct {
	Percentage        int32  `json:"percentage"`
	MinInMilliseconds *int64 `json:"min,omitempty"`
	MaxInMilliseconds *int64 `json:"max,omitempty"`
}

func (f *FaultInjectionDelay) Min(min *metav1.Duration) {
	if min != nil {
		f.MinInMilliseconds = ptr.To(min.Milliseconds())
	}
}

func (f *FaultInjectionDelay) Max(max *metav1.Duration) {
	if max != nil {
		f.MaxInMilliseconds = ptr.To(max.Milliseconds())
	}
}

// ---

type RateLimitSpec struct {
	Burst                  *int32                         `json:"burst,omitempty"`
	Requests               *int32                         `json:"requests,omitempty"`
	IntervalInMilliseconds *int64                         `json:"interval,omitempty"`
	Backlog                *int32                         `json:"backlog,omitempty"`
	RateLimitResponse      *extv1alpha1.RateLimitResponse `json:"response,omitempty"`
}

func (r *RateLimitSpec) Interval(interval *metav1.Duration) {
	if interval != nil {
		r.IntervalInMilliseconds = ptr.To(interval.Milliseconds())
	}
}

// ---

type HTTPLogSpec struct {
	Target      string            `json:"target"`
	Method      *string           `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	BufferLimit *int64            `json:"bufferLimit,omitempty"`
	Batch       *HTTPLogBatch     `json:"batch,omitempty"`
}

type HTTPLogBatch struct {
	Size                   *int32  `json:"size,omitempty"`
	IntervalInMilliseconds *int64  `json:"interval,omitempty"`
	Prefix                 *string `json:"prefix,omitempty"`
	Postfix                *string `json:"postfix,omitempty"`
	Separator              *string `json:"separator,omitempty"`
}

func (b *HTTPLogBatch) Interval(interval *metav1.Duration) {
	if interval != nil {
		b.IntervalInMilliseconds = ptr.To(interval.Milliseconds())
	}
}

// ---

type MetricsSpec struct {
	SampleIntervalInMilliseconds *int64 `json:"sampleInterval,omitempty"`
}

func (m *MetricsSpec) SampleInterval(sampleInterval *metav1.Duration) {
	if sampleInterval != nil {
		m.SampleIntervalInMilliseconds = ptr.To(sampleInterval.Milliseconds())
	}
}
