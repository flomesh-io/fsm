package status

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
)

// RouteInfo is a struct for storing route information
type RouteInfo struct {
	Meta       metav1.Object
	Parents    []gwv1beta1.RouteParentStatus
	GVK        schema.GroupVersionKind
	Generation int64
	Hostnames  []gwv1beta1.Hostname
}

var (
	defaultGatewayAPIObjectMapping = map[string]map[string]client.Object{
		constants.GatewayAPIGroup: {
			constants.GatewayAPIGatewayKind:   &gwv1beta1.Gateway{},
			constants.GatewayAPIHTTPRouteKind: &gwv1beta1.HTTPRoute{},
			constants.GatewayAPIGRPCRouteKind: &gwv1alpha2.GRPCRoute{},
		},
	}

	defaultServiceObjectMapping = map[string]map[string]client.Object{
		constants.KubernetesCoreGroup: {
			constants.KubernetesServiceKind: &corev1.Service{},
		},
		constants.FlomeshAPIGroup: {
			constants.FlomeshAPIServiceImportKind: &mcsv1alpha1.ServiceImport{},
		},
	}
)
