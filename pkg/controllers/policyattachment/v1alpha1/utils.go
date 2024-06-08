package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func getRouteParentKey(route metav1.Object, parentRef gwv1.ParentReference) types.NamespacedName {
	return types.NamespacedName{
		Namespace: gwutils.NamespaceDerefOr(parentRef.Namespace, route.GetNamespace()),
		Name:      string(parentRef.Name),
	}
}
