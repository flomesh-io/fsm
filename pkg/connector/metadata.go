package connector

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/lru"
)

// MicroSvcName defines string as microservice name
type MicroSvcName string

// MicroSvcDomainName defines string as microservice domain name
type MicroSvcDomainName string

// MicroEndpointAddr defines string as micro endpoint addr
type MicroEndpointAddr string

// To4 converts the IPv4 address ip to a 4-byte representation.
// If ip is not an IPv4 address, To4 returns nil.
func (addr MicroEndpointAddr) To4() net.IP {
	return net.ParseIP(string(addr)).To4()
}

// To16 converts the IP address ip to a 16-byte representation.
// If ip is not an IP address (it is the wrong length), To16 returns nil.
func (addr MicroEndpointAddr) To16() net.IP {
	return net.ParseIP(string(addr)).To16()
}

// MicroSvcPort defines int as micro service port
type MicroSvcPort int

// MicroSvcAppProtocol defines app protocol
type MicroSvcAppProtocol string

// MicroEndpointMeta defines micro endpoint meta
type MicroEndpointMeta struct {
	Ports   map[MicroSvcPort]MicroSvcAppProtocol
	Address MicroEndpointAddr

	Native struct {
		ClusterSet     string
		ClusterId      string
		ViaGatewayHTTP string `json:"ViaGatewayHttp,omitempty"`
		ViaGatewayGRPC string `json:"ViaGatewayGrpc,omitempty"`
		ViaGatewayMode ctv1.WithGatewayMode
	}
	Local struct {
		InternalService   bool
		WithGateway       bool
		WithMultiGateways bool
		BindFgwPorts      map[MicroSvcPort]MicroSvcAppProtocol
	}
}

func (m *MicroEndpointMeta) Init(controller ConnectController, discClient ServiceDiscoveryClient) {
	m.Local.InternalService = discClient.IsInternalServices()
	if len(m.Native.ClusterSet) > 0 &&
		!strings.EqualFold(m.Native.ClusterSet, controller.GetClusterSet()) {
		m.Local.InternalService = false
	}

	m.Local.WithGateway = controller.GetC2KWithGateway()
	m.Local.WithMultiGateways = controller.GetC2KMultiGateways()
	if !m.Local.WithGateway ||
		(len(m.Native.ViaGatewayHTTP) == 0 && len(m.Native.ViaGatewayGRPC) == 0) ||
		m.Local.InternalService {
		m.Local.WithMultiGateways = false
	}

	m.Local.BindFgwPorts = make(map[MicroSvcPort]MicroSvcAppProtocol)
	if m.Local.InternalService {
		if port := controller.GetViaIngressHTTPPort(); port > 0 {
			m.Local.BindFgwPorts[MicroSvcPort(port)] = constants.ProtocolHTTP
		}
		if port := controller.GetViaIngressGRPCPort(); port > 0 {
			m.Local.BindFgwPorts[MicroSvcPort(port)] = constants.ProtocolGRPC
		}
	} else {
		if port := controller.GetViaEgressHTTPPort(); port > 0 {
			m.Local.BindFgwPorts[MicroSvcPort(port)] = constants.ProtocolHTTP
		}
		if port := controller.GetViaEgressGRPCPort(); port > 0 {
			m.Local.BindFgwPorts[MicroSvcPort(port)] = constants.ProtocolGRPC
		}
	}
}

// MicroSvcMeta defines micro service meta
type MicroSvcMeta struct {
	Ports       map[MicroSvcPort]MicroSvcAppProtocol
	Endpoints   map[MicroEndpointAddr]*MicroEndpointMeta
	HealthCheck bool
}

func (m *MicroSvcMeta) Unmarshal(str string) {
	_ = json.Unmarshal([]byte(str), m)
}

func (m *MicroSvcMeta) Marshal() string {
	if bytes, err := json.Marshal(m); err == nil {
		return string(bytes)
	}
	return ""
}

func Encode(m *MicroSvcMeta) (string, uint64) {
	if bytes, err := json.Marshal(m); err == nil {
		h := fnv.New64()
		_, _ = h.Write(bytes)
		return base64.StdEncoding.EncodeToString(bytes), h.Sum64()
	}
	return "", 0
}

func Decode(svc *corev1.Service, enc string) *MicroSvcMeta {
	hash := svc.Annotations[constants.AnnotationMeshEndpointHash]
	key := fmt.Sprintf("%s.%s.%s", svc.Namespace, svc.Name, hash)
	if meta, ok := lru.Get(key); ok {
		lru.Add(key, meta)
		return meta.(*MicroSvcMeta)
	}
	meta := new(MicroSvcMeta)
	if bytes, err := base64.StdEncoding.DecodeString(enc); err == nil {
		if err = json.Unmarshal(bytes, meta); err == nil {
			lru.Add(key, meta)
		}
	}
	return meta
}
