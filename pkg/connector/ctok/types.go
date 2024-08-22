// Package ctok contains a reusable abstraction for efficiently
// watching for changes in resources in a Kubernetes cluster.
package ctok

import (
	"context"

	"github.com/flomesh-io/fsm/pkg/connector"
)

const (
	// CloudSourcedServiceLabel defines cloud-sourced service label
	CloudSourcedServiceLabel = "fsm-connector-cloud-sourced-service"
	// CloudServiceLabel defines cloud service label
	CloudServiceLabel = "fsm-connector-cloud-service"
)

// Aggregator aggregates micro services
type Aggregator interface {
	// Aggregate micro services
	Aggregate(context.Context, connector.MicroSvcName) map[connector.MicroSvcName]*connector.MicroSvcMeta
}
