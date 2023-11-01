package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func getTargetNamespace(owner client.Object, ref gwv1alpha2.PolicyTargetReference) string {
	if ref.Namespace == nil {
		return owner.GetNamespace()
	}

	return string(*ref.Namespace)
}

func getRouteParentKey(route metav1.Object, parent gwv1beta1.RouteParentStatus) types.NamespacedName {
	key := types.NamespacedName{Name: string(parent.ParentRef.Name), Namespace: route.GetNamespace()}
	if parent.ParentRef.Namespace != nil {
		key.Namespace = string(*parent.ParentRef.Namespace)
	}

	return key
}
