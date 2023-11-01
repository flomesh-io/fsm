package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type routeInfo struct {
	meta       metav1.Object
	parents    []gwv1beta1.RouteParentStatus
	gvk        schema.GroupVersionKind
	generation int64
	hostnames  []gwv1beta1.Hostname
}
