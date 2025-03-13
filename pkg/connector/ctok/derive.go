package ctok

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
)

// IsSyncCloudNamespace if sync namespace
func IsSyncCloudNamespace(ns *corev1.Namespace) bool {
	if ns != nil {
		if len(ns.Annotations) > 0 && len(ns.Labels) > 0 {
			_, meshMonitored := ns.Labels[constants.FSMKubeResourceMonitorAnnotation]
			_, meshServiceSync := ns.Annotations[connector.AnnotationMeshServiceSync]
			return meshMonitored && meshServiceSync
		}
	}
	return false
}

func (s *CtoKSyncer) hasOwnership(service *corev1.Service) bool {
	return len(service.Labels) > 0 && len(service.Annotations) > 0 &&
		service.Labels[constants.CloudSourcedServiceLabel] == True &&
		service.Annotations[connector.AnnotationMeshServiceSyncManagedBy] == s.controller.GetConnectorUID()
}
