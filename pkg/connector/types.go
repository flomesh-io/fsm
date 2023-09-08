// Package connector contains a reusable abstraction for efficiently
// watching for changes in resources in a Kubernetes cluster.
package connector

const (
	// CloudSourcedServiceLabel defines cloud-sourced service label
	CloudSourcedServiceLabel = "cloud-sourced-service"
	// CloudServiceLabel defines cloud service label
	CloudServiceLabel = "cloud-service"
	// CloudServiceInheritedFromAnnotation defines cloud service inherited annotation
	CloudServiceInheritedFromAnnotation = "flomesh.io/cloud-service-inherited-from"

	// MeshServiceSyncAnnotation defines mesh service sync annotation
	MeshServiceSyncAnnotation = "flomesh.io/mesh-service-sync"
	// MeshEndpointAddrAnnotation defines mesh endpoint addr annotation
	MeshEndpointAddrAnnotation = "flomesh.io/cloud-endpoint-addr"
)

// MicroSvcName defines string as microservice name
type MicroSvcName string

// MicroSvcDomainName defines string as microservice domain name
type MicroSvcDomainName string

// MicroEndpointAddr defines string as micro endpoint addr
type MicroEndpointAddr string

// MicroSvcPort defines int as micro service port
type MicroSvcPort int

// MicroSvcAppProtocol defines app protocol
type MicroSvcAppProtocol string

// MicroSvcMeta defines micro service meta
type MicroSvcMeta struct {
	Ports     map[MicroSvcPort]MicroSvcAppProtocol
	Addresses map[MicroEndpointAddr]int
}

// Aggregator aggregates micro services
type Aggregator interface {
	// Aggregate micro services
	Aggregate(svcName MicroSvcName, svcDomainName MicroSvcDomainName) (map[MicroSvcName]*MicroSvcMeta, string)
}
