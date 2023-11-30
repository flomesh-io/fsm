package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func getRouteParentKey(route metav1.Object, parent gwv1beta1.RouteParentStatus) types.NamespacedName {
	return types.NamespacedName{
		Namespace: gwutils.Namespace(parent.ParentRef.Namespace, route.GetNamespace()),
		Name:      string(parent.ParentRef.Name),
	}
}
