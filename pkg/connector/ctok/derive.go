package ctok

import (
	apiv1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
)

// IsSyncCloudNamespace if sync namespace
func IsSyncCloudNamespace(ns *apiv1.Namespace) bool {
	if ns != nil {
		_, exists := ns.Annotations[connector.AnnotationMeshServiceSync]
		return exists
	}
	return false
}

func (s *CtoKSyncer) hasOwnership(service *apiv1.Service) bool {
	return len(service.Labels) > 0 && len(service.Annotations) > 0 &&
		service.Labels[constants.CloudSourcedServiceLabel] == True &&
		service.Annotations[connector.AnnotationMeshServiceSyncManagedBy] == s.controller.GetConnectorUID()
}
