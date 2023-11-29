package ktoc

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
)

var (
	syncCloudNamespace string
)

// SetSyncCloudNamespace sets sync namespace
func SetSyncCloudNamespace(ns string) {
	syncCloudNamespace = ns
}

// IsSyncCloudNamespace if sync namespace
func IsSyncCloudNamespace(ns *corev1.Namespace) bool {
	if ns != nil {
		_, exists := ns.Annotations[connector.AnnotationMeshServiceSync]
		return exists
	}
	return false
}
