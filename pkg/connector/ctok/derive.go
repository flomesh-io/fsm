package ctok

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
)

// IsSyncCloudNamespace if sync namespace
func IsSyncCloudNamespace(ns *corev1.Namespace) bool {
	if ns != nil {
		_, exists := ns.Annotations[connector.AnnotationMeshServiceSync]
		return exists
	}
	return false
}
