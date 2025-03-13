// Package ctok contains a reusable abstraction for efficiently
// watching for changes in resources in a Kubernetes cluster.
package ctok

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("connector-c2k")
)

// Aggregator aggregates micro services
type Aggregator interface {
	// Aggregate micro services
	Aggregate(ctx context.Context, kubeSvcName connector.KubeSvcName) map[connector.KubeSvcName]*connector.MicroSvcMeta
}

type syncCreate struct {
	service   *corev1.Service
	endpoints *corev1.Endpoints
}

type SharedServiceList struct {
}
