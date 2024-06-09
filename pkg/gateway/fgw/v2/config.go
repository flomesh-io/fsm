package v2

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type Config struct {
	Gateway     *Gateway          `json:"gateway"`
	Resources   []interface{}     `json:"resources" hash:"set"`
	SecretFiles map[string]string `json:"secretFiles"`
	Version     string            `json:"version" hash:"ignore"`
}

type Gateway struct {
	Kind      string      `json:"kind,omitempty"`
	Namespace string      `json:"namespace"`
	Name      string      `json:"name"`
	Spec      GatewaySpec `json:"spec"`
}

type GatewaySpec struct {
	GatewayClassName gwv1.ObjectName       `json:"gatewayClassName"`
	Listeners        []Listener            `json:"listeners" copier:"-"`
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
	CertChain  string `json:"certChain,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

type HTTPRoute struct {
	Kind      string        `json:"kind,omitempty"`
	Namespace string        `json:"namespace"`
	Name      string        `json:"name"`
	Spec      HTTPRouteSpec `json:"spec"`
}

type HTTPRouteSpec struct {
	gwv1.CommonRouteSpec `json:",inline"`
	Hostnames            []gwv1.Hostname `json:"hostnames,omitempty"`
	Rules                []HTTPRouteRule `json:"rules,omitempty" copier:"-"`
}
type HTTPRouteRule struct {
	Matches            []gwv1.HTTPRouteMatch    `json:"matches,omitempty"`
	Filters            []gwv1.HTTPRouteFilter   `json:"filters,omitempty"`
	BackendRefs        []gwv1.HTTPBackendRef    `json:"backendRefs,omitempty" copier:"-"`
	Timeouts           *gwv1.HTTPRouteTimeouts  `json:"timeouts,omitempty"`
	SessionPersistence *gwv1.SessionPersistence `json:"sessionPersistence,omitempty"`
}

type GRPCRoute struct {
	Kind      string        `json:"kind,omitempty"`
	Namespace string        `json:"namespace"`
	Name      string        `json:"name"`
	Spec      GRPCRouteSpec `json:"spec,omitempty"`
}

type GRPCRouteSpec struct {
	gwv1.CommonRouteSpec `json:",inline"`
	Hostnames            []gwv1.Hostname `json:"hostnames,omitempty"`
	Rules                []GRPCRouteRule `json:"rules,omitempty" copier:"-"`
}

type GRPCRouteRule struct {
	Matches            []gwv1.GRPCRouteMatch    `json:"matches,omitempty"`
	Filters            []gwv1.GRPCRouteFilter   `json:"filters,omitempty"`
	BackendRefs        []gwv1.GRPCBackendRef    `json:"backendRefs,omitempty" copier:"-"`
	SessionPersistence *gwv1.SessionPersistence `json:"sessionPersistence,omitempty"`
}

type TCPRoute struct {
	Kind      string       `json:"kind,omitempty"`
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	Spec      TCPRouteSpec `json:"spec"`
}

// TCPRouteSpec defines the desired state of TCPRoute
type TCPRouteSpec struct {
	gwv1alpha2.CommonRouteSpec `json:",inline"`
	Rules                      []TCPRouteRule `json:"rules" copier:"-"`
}

type TCPRouteRule struct {
	BackendRefs []gwv1alpha2.BackendRef `json:"backendRefs,omitempty" copier:"-"`
}

type TLSRoute struct {
	Kind      string                  `json:"kind,omitempty"`
	Namespace string                  `json:"namespace"`
	Name      string                  `json:"name"`
	Spec      gwv1alpha2.TLSRouteSpec `json:"spec"`
}

type UDPRoute struct {
	Kind      string       `json:"kind,omitempty"`
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	Spec      UDPRouteSpec `json:"spec"`
}

type UDPRouteSpec struct {
	gwv1alpha2.CommonRouteSpec `json:",inline"`
	Rules                      []UDPRouteRule `json:"rules" copier:"-"`
}

type UDPRouteRule struct {
	BackendRefs []gwv1alpha2.BackendRef `json:"backendRefs,omitempty" copier:"-"`
}
