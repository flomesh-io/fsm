// Package ctok contains a reusable abstraction for efficiently
// watching for changes in resources in a Kubernetes cluster.
package ctok

import "context"

const (
	// CloudSourcedServiceLabel defines cloud-sourced service label
	CloudSourcedServiceLabel = "cloud-sourced-service"
	// CloudServiceLabel defines cloud service label
	CloudServiceLabel = "cloud-service"
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
	Ports       map[MicroSvcPort]MicroSvcAppProtocol
	Addresses   map[MicroEndpointAddr]int
	ClusterId   string
	HealthCheck bool
}

// Aggregator aggregates micro services
type Aggregator interface {
	// Aggregate micro services
	Aggregate(context.Context, MicroSvcName) map[MicroSvcName]*MicroSvcMeta
}
