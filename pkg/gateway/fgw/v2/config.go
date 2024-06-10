package v2

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type Config struct {
	//Gateway     *Gateway          `json:"gateway"`
	Resources []interface{}     `json:"resources" hash:"set"`
	Secrets   map[string]string `json:"secrets"`
	Version   string            `json:"version" hash:"ignore"`
}

type ObjectMeta struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

type Gateway struct {
	Kind       string      `json:"kind"`
	ObjectMeta ObjectMeta  `json:"metadata"`
	Spec       GatewaySpec `json:"spec"`
}

type GatewaySpec struct {
	GatewayClassName gwv1.ObjectName       `json:"gatewayClassName"`
	Listeners        []Listener            `json:"listeners,omitempty" copier:"-"`
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
	Certificates       []map[string]string                         `json:"certificates,omitempty" copier:"-"`
	FrontendValidation *FrontendTLSValidation                      `json:"frontendValidation,omitempty" copier:"-"`
	Options            map[gwv1.AnnotationKey]gwv1.AnnotationValue `json:"options,omitempty"`
}

type FrontendTLSValidation struct {
	CACertificates []map[string]string `json:"caCertificates,omitempty" copier:"-"`
}

// Certificate is the certificate configuration
type Certificate struct {
	CertChain  string `json:"certChain"`
	PrivateKey string `json:"privateKey"`
}

type HTTPRoute struct {
	Kind       string        `json:"kind"`
	ObjectMeta ObjectMeta    `json:"metadata"`
	Spec       HTTPRouteSpec `json:"spec"`
}

type HTTPRouteSpec struct {
	gwv1.CommonRouteSpec `json:",inline"`
	Hostnames            []gwv1.Hostname `json:"hostnames,omitempty"`
	Rules                []HTTPRouteRule `json:"rules,omitempty" copier:"-"`
}
type HTTPRouteRule struct {
	Matches            []gwv1.HTTPRouteMatch    `json:"matches,omitempty"`
	Filters            []HTTPRouteFilter        `json:"filters,omitempty"`
	BackendRefs        []HTTPBackendRef         `json:"backendRefs,omitempty" copier:"-"`
	Timeouts           *gwv1.HTTPRouteTimeouts  `json:"timeouts,omitempty"`
	SessionPersistence *gwv1.SessionPersistence `json:"sessionPersistence,omitempty"`
}

type GRPCRoute struct {
	Kind       string        `json:"kind"`
	ObjectMeta ObjectMeta    `json:"metadata"`
	Spec       GRPCRouteSpec `json:"spec,omitempty"`
}

type GRPCRouteSpec struct {
	gwv1.CommonRouteSpec `json:",inline"`
	Hostnames            []gwv1.Hostname `json:"hostnames,omitempty"`
	Rules                []GRPCRouteRule `json:"rules,omitempty" copier:"-"`
}

type GRPCRouteRule struct {
	Matches            []gwv1.GRPCRouteMatch    `json:"matches,omitempty"`
	Filters            []GRPCRouteFilter        `json:"filters,omitempty"`
	BackendRefs        []GRPCBackendRef         `json:"backendRefs,omitempty" copier:"-"`
	SessionPersistence *gwv1.SessionPersistence `json:"sessionPersistence,omitempty"`
}

type TCPRoute struct {
	Kind       string       `json:"kind"`
	ObjectMeta ObjectMeta   `json:"metadata"`
	Spec       TCPRouteSpec `json:"spec"`
}

// TCPRouteSpec defines the desired state of TCPRoute
type TCPRouteSpec struct {
	gwv1alpha2.CommonRouteSpec `json:",inline"`
	Rules                      []TCPRouteRule `json:"rules" copier:"-"`
}

type TCPRouteRule struct {
	BackendRefs []BackendRef `json:"backendRefs,omitempty" copier:"-"`
}

type TLSRoute struct {
	Kind       string       `json:"kind"`
	ObjectMeta ObjectMeta   `json:"metadata"`
	Spec       TLSRouteSpec `json:"spec"`
}

// TLSRouteSpec defines the desired state of a TLSRoute resource.
type TLSRouteSpec struct {
	gwv1alpha2.CommonRouteSpec `json:",inline"`
	Hostnames                  []gwv1alpha2.Hostname `json:"hostnames,omitempty"`
	Rules                      []TLSRouteRule        `json:"rules" copier:"-"`
}

type TLSRouteRule struct {
	BackendRefs []BackendRef `json:"backendRefs,omitempty"`
}

type UDPRoute struct {
	Kind       string       `json:"kind"`
	ObjectMeta ObjectMeta   `json:"metadata"`
	Spec       UDPRouteSpec `json:"spec"`
}

type UDPRouteSpec struct {
	gwv1alpha2.CommonRouteSpec `json:",inline"`
	Rules                      []UDPRouteRule `json:"rules" copier:"-"`
}

type UDPRouteRule struct {
	BackendRefs []BackendRef `json:"backendRefs,omitempty" copier:"-"`
}

type HTTPBackendRef struct {
	Kind    string            `json:"kind"`
	Name    string            `json:"name"`
	Weight  int32             `json:"weight,omitempty"`
	Filters []HTTPRouteFilter `json:"filters,omitempty"`
}

type HTTPRouteFilter struct {
	Type                   gwv1.HTTPRouteFilterType        `json:"type"`
	RequestHeaderModifier  *gwv1.HTTPHeaderFilter          `json:"requestHeaderModifier,omitempty"`
	ResponseHeaderModifier *gwv1.HTTPHeaderFilter          `json:"responseHeaderModifier,omitempty"`
	RequestMirror          *HTTPRequestMirrorFilter        `json:"requestMirror,omitempty"`
	RequestRedirect        *gwv1.HTTPRequestRedirectFilter `json:"requestRedirect,omitempty"`
	URLRewrite             *gwv1.HTTPURLRewriteFilter      `json:"urlRewrite,omitempty"`
	ExtensionRef           *gwv1.LocalObjectReference      `json:"extensionRef,omitempty"`
}

type HTTPRequestMirrorFilter struct {
	BackendRef BackendRef `json:"backendRef"`
}

type GRPCBackendRef struct {
	Kind    string            `json:"kind"`
	Name    string            `json:"name"`
	Weight  int32             `json:"weight,omitempty"`
	Filters []GRPCRouteFilter `json:"filters,omitempty"`
}

type GRPCRouteFilter struct {
	Type                   gwv1.GRPCRouteFilterType   `json:"type"`
	RequestHeaderModifier  *gwv1.HTTPHeaderFilter     `json:"requestHeaderModifier,omitempty"`
	ResponseHeaderModifier *gwv1.HTTPHeaderFilter     `json:"responseHeaderModifier,omitempty"`
	RequestMirror          *HTTPRequestMirrorFilter   `json:"requestMirror,omitempty"`
	ExtensionRef           *gwv1.LocalObjectReference `json:"extensionRef,omitempty"`
}

type BackendRef struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Weight int32  `json:"weight,omitempty"`
}

type Backend struct {
	Kind       string      `json:"kind"`
	ObjectMeta ObjectMeta  `json:"metadata"`
	Spec       BackendSpec `json:"spec"`
	Port       int32       `json:"-"` // store the port for the backend temporarily
}

type BackendSpec struct {
	Targets []BackendTarget `json:"targets,omitempty"`
}

type BackendTarget struct {
	Address string            `json:"address"`
	Port    *int32            `json:"port"`
	Weight  int32             `json:"weight,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
}

// ServicePortName is a combination of a service name, namespace, and port
type ServicePortName struct {
	types.NamespacedName
	Port *int32
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
