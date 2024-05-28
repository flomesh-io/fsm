package utils

import (
	"github.com/flomesh-io/fsm/pkg/constants"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func IsValidTargetRefToGroupKindOfService(ref gwv1alpha2.NamespacedPolicyTargetReference) bool {
	if (ref.Kind == constants.KubernetesServiceKind && ref.Group == constants.KubernetesCoreGroup) ||
		(ref.Kind == constants.FlomeshAPIServiceImportKind && ref.Group == constants.FlomeshMCSAPIGroup) {
		return true
	}

	return false
}
