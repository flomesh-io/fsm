// Package connector contains a reusable abstraction for efficiently
// watching for changes in resources in a Kubernetes cluster.
package connector

const (
	// CloudSourcedServiceLabel defines cloud-sourced service label
	CloudSourcedServiceLabel = "cloud-sourced-service"
	// CloudServiceLabel defines cloud service label
	CloudServiceLabel = "cloud-service"

	// MeshServiceSyncAnnotation defines mesh service sync annotation
	MeshServiceSyncAnnotation = "flomesh.io/mesh-service-sync"
	// MeshEndpointSddrAnnotation defines mesh endpoint addr annotation
	MeshEndpointSddrAnnotation = "flomesh.io/cloud-endpoint-addr"
)

// MicroSvcName defines string as microservice name
type MicroSvcName string

// MicroSvcDomainName defines string as microservice domain name
type MicroSvcDomainName string

// MicroEndpointAddr defines string as micro endpoint addr
type MicroEndpointAddr string

// MicroSvcPort defines int as micro service port
type MicroSvcPort struct {
	Name        string
	Port        int
	AppProtocol string
}

// Aggregator aggregates micro services
type Aggregator interface {
	// Aggregate micro services
	Aggregate(svcName MicroSvcName, svcDomainName MicroSvcDomainName) ([]MicroSvcName, []MicroSvcPort, []MicroEndpointAddr)
}
