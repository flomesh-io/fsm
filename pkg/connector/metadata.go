package connector

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/lru"
)

// KubeSvcKey is in the form <kube namespace>/<kube svc name>
type KubeSvcKey string

type KubeSvcName string

type CloudSvcName string

type ExternalName string

type ServiceConversion struct {
	Service      CloudSvcName
	ExternalName ExternalName
}

// MicroEndpointMeta defines micro endpoint meta
type MicroEndpointMeta struct {
	Ports   map[MicroServicePort]MicroServiceProtocol `json:"ports,omitempty"`
	Address MicroServiceAddress                       `json:"address,omitempty"`

	GRPCMeta map[string]interface{} `json:"grpcMeta,omitempty"`

	Native struct {
		ClusterSet     string               `json:"clusterSet,omitempty"`
		ClusterId      string               `json:"clusterId,omitempty"`
		ViaGatewayHTTP string               `json:"viaGatewayHttp,omitempty"`
		ViaGatewayGRPC string               `json:"viaGatewayGrpc,omitempty"`
		ViaGatewayMode ctv1.WithGatewayMode `json:"viaGatewayMode,omitempty"`
	} `json:"native"`
	Local struct {
		InternalService   bool                                      `json:"internalService,omitempty"`
		WithGateway       bool                                      `json:"withGateway,omitempty"`
		WithMultiGateways bool                                      `json:"withMultiGateways,omitempty"`
		BindFgwPorts      map[MicroServicePort]MicroServiceProtocol `json:"bindFgwPorts,omitempty"`
	} `json:"local"`
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

	m.Local.BindFgwPorts = make(map[MicroServicePort]MicroServiceProtocol)
	if m.Local.InternalService {
		if port := controller.GetViaIngressHTTPPort(); port > 0 {
			m.Local.BindFgwPorts[MicroServicePort(port)] = constants.ProtocolHTTP
		}
		if port := controller.GetViaIngressGRPCPort(); port > 0 {
			m.Local.BindFgwPorts[MicroServicePort(port)] = constants.ProtocolGRPC
		}
	} else {
		if port := controller.GetViaEgressHTTPPort(); port > 0 {
			m.Local.BindFgwPorts[MicroServicePort(port)] = constants.ProtocolHTTP
		}
		if port := controller.GetViaEgressGRPCPort(); port > 0 {
			m.Local.BindFgwPorts[MicroServicePort(port)] = constants.ProtocolGRPC
		}
	}
}

type GRPCMeta struct {
	Interface string              `json:"interface,omitempty"`
	Methods   map[string][]string `json:"methods,omitempty"`
}

// MicroSvcMeta defines micro service meta
type MicroSvcMeta struct {
	Ports       map[MicroServicePort]MicroServicePort      `json:"-"`
	TargetPorts map[MicroServicePort]MicroServiceProtocol  `json:"ports,omitempty"`
	Endpoints   map[MicroServiceAddress]*MicroEndpointMeta `json:"endpoints,omitempty"`

	GRPCMeta *GRPCMeta `json:"grpcMeta,omitempty"`

	HealthCheck bool `json:"healthcheck,omitempty"`
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
