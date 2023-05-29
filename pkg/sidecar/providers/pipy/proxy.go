package pipy

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/models"
	"github.com/flomesh-io/fsm/pkg/sidecar"
)

// Proxy is a representation of an Sidecar proxy .
// This should at some point have a 1:1 match to an Endpoint (which is a member of a meshed service).
type Proxy struct {

	// UUID of the proxy
	uuid.UUID

	Identity identity.ServiceIdentity

	net.Addr

	// The time this Proxy connected to the FSM control plane
	connectedAt time.Time

	// kind is the proxy's kind (ex. sidecar, gateway)
	kind models.ProxyKind

	// Records metadata around the Kubernetes Pod on which this Sidecar Proxy is installed.
	// This could be nil if the Sidecar is not operating in a Kubernetes cluster (VM for example)
	// NOTE: This field may be not be set at the time Proxy struct is initialized. This would
	// eventually be set when the metadata arrives via the xDS protocol.
	PodMetadata *PodMetadata

	MeshConf    *configurator.Configurator
	SidecarCert *certificate.Certificate

	// The version of Pipy Repo Codebase
	ETag uint64

	Mutex *sync.RWMutex
	Quit  chan bool

	ID uint64
}

func (p *Proxy) String() string {
	return fmt.Sprintf("[ProxyUUID=%s], [Pod metadata=%s]", p.UUID, p.PodMetadataString())
}

// PodMetadata is a struct holding information on the Pod on which a given Sidecar proxy is installed
// This struct is initialized *eventually*, when the metadata arrives via xDS.
type PodMetadata struct {
	UID             string
	Name            string
	Namespace       string
	IP              string
	ServiceAccount  identity.K8sServiceAccount
	CreationTime    time.Time
	Cluster         string
	SidecarNodeID   string
	WorkloadKind    string
	WorkloadName    string
	ReadinessProbes []*v1.Probe
	LivenessProbes  []*v1.Probe
	StartupProbes   []*v1.Probe
}

// HasPodMetadata answers the question - has the Pod metadata been recorded for the given Sidecar proxy
func (p *Proxy) HasPodMetadata() bool {
	return p.PodMetadata != nil
}

// StatsHeaders returns the headers required for SMI metrics
func (p *Proxy) StatsHeaders() map[string]string {
	unknown := "unknown"
	podName := unknown
	podNamespace := unknown
	podControllerKind := unknown
	podControllerName := unknown

	if p.PodMetadata != nil {
		if len(p.PodMetadata.Name) > 0 {
			podName = p.PodMetadata.Name
		}
		if len(p.PodMetadata.Namespace) > 0 {
			podNamespace = p.PodMetadata.Namespace
		}
		if len(p.PodMetadata.WorkloadKind) > 0 {
			podControllerKind = p.PodMetadata.WorkloadKind
		}
		if len(p.PodMetadata.WorkloadName) > 0 {
			podControllerName = p.PodMetadata.WorkloadName
		}
	}

	// Assume ReplicaSets are controlled by a Deployment unless their names
	// do not contain a hyphen. This aligns with the behavior of the
	// Prometheus config in the FSM Helm chart.
	if podControllerKind == "ReplicaSet" {
		if hyp := strings.LastIndex(podControllerName, "-"); hyp >= 0 {
			podControllerKind = "Deployment"
			podControllerName = podControllerName[:hyp]
		}
	}

	return map[string]string{
		"fsm-stats-pod":       podName,
		"fsm-stats-namespace": podNamespace,
		"fsm-stats-kind":      podControllerKind,
		"fsm-stats-name":      podControllerName,
	}
}

// PodMetadataString returns relevant pod metadata as a string
func (p *Proxy) PodMetadataString() string {
	if p.PodMetadata == nil {
		return ""
	}
	return fmt.Sprintf("UID=%s, Namespace=%s, Name=%s, ServiceAccount=%s", p.PodMetadata.UID, p.PodMetadata.Namespace, p.PodMetadata.Name, p.PodMetadata.ServiceAccount.Name)
}

// GetName returns a unique name for this proxy based on the identity and uuid.
func (p *Proxy) GetName() string {
	return fmt.Sprintf("%s:%s", p.Identity.String(), p.UUID.String())
}

// GetUUID returns UUID.
func (p *Proxy) GetUUID() uuid.UUID {
	return p.UUID
}

// GetIdentity returns ServiceIdentity.
func (p *Proxy) GetIdentity() identity.ServiceIdentity {
	return p.Identity
}

// GetConnectedAt returns the timestamp of when the given proxy connected to the control plane.
func (p *Proxy) GetConnectedAt() time.Time {
	return p.connectedAt
}

// GetIP returns the address of the proxy connected.
func (p *Proxy) GetIP() net.Addr {
	return p.Addr
}

// GetAddr returns the IP address of the proxy connected.
func (p *Proxy) GetAddr() string {
	if p.Addr == nil {
		return ""
	}
	return p.Addr.String()
}

// Kind return the proxy's kind
func (p *Proxy) Kind() models.ProxyKind {
	return p.kind
}

// GetCNPrefix returns a newly generated CommonName for a certificate of the form: <ProxyUUID>.<kind>.<identity>
// where identity itself is of the form <name>.<namespace>
func (p *Proxy) GetCNPrefix() string {
	return sidecar.GetCertCNPrefix(p, models.KindSidecar)
}

var (
	proxyLock sync.Mutex
	proxyID   = uint64(0)
)

// NewProxy creates a new instance of an Sidecar proxy connected to the servers.
func NewProxy(kind models.ProxyKind, uuid uuid.UUID, svcIdentity identity.ServiceIdentity, ip net.Addr) *Proxy {
	proxyLock.Lock()
	proxyID++
	id := proxyID
	defer proxyLock.Unlock()
	return &Proxy{
		// Identity is of the form <name>.<namespace>.cluster.local
		Identity:    svcIdentity,
		UUID:        uuid,
		Addr:        ip,
		connectedAt: time.Now(),
		kind:        kind,
		Mutex:       new(sync.RWMutex),
		ID:          id,
	}
}
