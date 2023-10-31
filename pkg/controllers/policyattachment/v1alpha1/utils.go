package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func getTargetNamespace(owner client.Object, ref gwv1alpha2.PolicyTargetReference) string {
	if ref.Namespace == nil {
		return owner.GetNamespace()
	}

	return string(*ref.Namespace)
}
