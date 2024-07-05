package fgw

import (
	"fmt"

	"k8s.io/utils/ptr"

	"github.com/google/go-cmp/cmp"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type ConfigSpec struct {
	Resources []interface{}                  `json:"resources" hash:"set"`
	Secrets   map[string]string              `json:"secrets"`
	Filters   map[string][]map[string]string `json:"filters"`
	Version   string                         `json:"version" hash:"ignore"`
}

func (c *ConfigSpec) GetVersion() string {
	return c.Version
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

type Gateway struct {
	Kind       string      `json:"kind"`
	ObjectMeta ObjectMeta  `json:"metadata"`
	Spec       GatewaySpec `json:"spec"`
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
	Kind       string        `json:"kind"`
	ObjectMeta ObjectMeta    `json:"metadata"`
	Spec       HTTPRouteSpec `json:"spec"`
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
	Kind       string        `json:"kind"`
	ObjectMeta ObjectMeta    `json:"metadata"`
	Spec       GRPCRouteSpec `json:"spec,omitempty"`
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
	Kind       string       `json:"kind"`
	ObjectMeta ObjectMeta   `json:"metadata"`
	Spec       TCPRouteSpec `json:"spec"`
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
	Kind       string       `json:"kind"`
	ObjectMeta ObjectMeta   `json:"metadata"`
	Spec       TLSRouteSpec `json:"spec"`
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
	Kind       string       `json:"kind"`
	ObjectMeta ObjectMeta   `json:"metadata"`
	Spec       UDPRouteSpec `json:"spec"`
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
	ExtensionRef           *gwv1.LocalObjectReference      `json:"extensionRef,omitempty"`
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
	Type                   gwv1.GRPCRouteFilterType   `json:"type"`
	RequestHeaderModifier  *gwv1.HTTPHeaderFilter     `json:"requestHeaderModifier,omitempty"`
	ResponseHeaderModifier *gwv1.HTTPHeaderFilter     `json:"responseHeaderModifier,omitempty"`
	RequestMirror          *HTTPRequestMirrorFilter   `json:"requestMirror,omitempty"`
	ExtensionRef           *gwv1.LocalObjectReference `json:"extensionRef,omitempty"`
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
	Kind       string      `json:"kind"`
	ObjectMeta ObjectMeta  `json:"metadata"`
	Spec       BackendSpec `json:"spec"`
	Port       int32       `json:"-"` // store the port for the backend temporarily
}

func NewBackend(svcPortName string, targets []BackendTarget) *Backend {
	return &Backend{
		Kind: "Backend",
		ObjectMeta: ObjectMeta{
			Name: svcPortName,
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
	Kind       string               `json:"kind"`
	ObjectMeta ObjectMeta           `json:"metadata"`
	Spec       BackendTLSPolicySpec `json:"spec"`
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
	Kind       string              `json:"kind"`
	ObjectMeta ObjectMeta          `json:"metadata"`
	Spec       BackendLBPolicySpec `json:"spec"`
}

type BackendLBPolicySpec struct {
	TargetRefs         []BackendRef             `json:"targetRefs" copier:"-" hash:"set"`
	SessionPersistence *gwv1.SessionPersistence `json:"sessionPersistence,omitempty"`
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
	Kind       string          `json:"kind"`
	ObjectMeta ObjectMeta      `json:"metadata"`
	Spec       RetryPolicySpec `json:"spec"`
}

type RetryPolicySpec struct {
	TargetRefs   []BackendRef              `json:"targetRefs" copier:"-" hash:"set"`
	Ports        []gwpav1alpha2.PortRetry  `json:"ports,omitempty" hash:"set"`
	DefaultRetry *gwpav1alpha2.RetryConfig `json:"retry,omitempty"`
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
			p.Spec.Ports = append(p.Spec.Ports, port)
		}
	} else {
		p.Spec.Ports = []gwpav1alpha2.PortRetry{port}
	}
}

// ---

type HealthCheckPolicy struct {
	Kind       string                `json:"kind"`
	ObjectMeta ObjectMeta            `json:"metadata"`
	Spec       HealthCheckPolicySpec `json:"spec"`
}

type HealthCheckPolicySpec struct {
	TargetRefs         []BackendRef                    `json:"targetRefs" copier:"-" hash:"set"`
	Ports              []gwpav1alpha2.PortHealthCheck  `json:"ports,omitempty" copier:"-" hash:"set"`
	DefaultHealthCheck *gwpav1alpha2.HealthCheckConfig `json:"healthCheck,omitempty"`
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
			p.Spec.Ports = append(p.Spec.Ports, port)
		}
	} else {
		p.Spec.Ports = []gwpav1alpha2.PortHealthCheck{port}
	}
}
