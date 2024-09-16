// Package policy implements the Kubernetes client for the resources in the policy.flomesh.io API group
package policy

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/service"
)

var (
	log = logger.New("policy-controller")
)

// Client is the type used to represent the Kubernetes Client for the policy.flomesh.io API group
type Client struct {
	informers      *informers.InformerCollection
	kubeClient     kubernetes.Interface
	kubeController k8s.Controller
}

// Controller is the interface for the functionality provided by the resources part of the policy.flomesh.io API group
type Controller interface {
	// ListIsolationPolicies returns the Isolation policies
	ListIsolationPolicies() []*policyv1alpha1.Isolation

	// ListEgressGateways lists Egress gateways
	ListEgressGateways() []*policyv1alpha1.EgressGateway

	// ListEgressPoliciesForSourceIdentity lists the Egress policies for the given source identity
	ListEgressPoliciesForSourceIdentity(identity.K8sServiceAccount) []*policyv1alpha1.Egress

	// GetEgressSourceSecret returns the secret resource that matches the given options
	GetEgressSourceSecret(corev1.SecretReference) (*corev1.Secret, error)

	// GetIngressBackendPolicy returns the IngressBackend policy for the given backend MeshService
	GetIngressBackendPolicy(service.MeshService) *policyv1alpha1.IngressBackend

	// ListRetryPolicies returns the Retry policies for the given source identity
	ListRetryPolicies(identity.K8sServiceAccount) []*policyv1alpha1.Retry

	// GetAccessControlPolicy returns the AccessControl policy for the given backend MeshService
	GetAccessControlPolicy(service.MeshService) *policyv1alpha1.AccessControl

	// GetUpstreamTrafficSetting returns the UpstreamTrafficSetting resource that matches the given options
	GetUpstreamTrafficSetting(UpstreamTrafficSettingGetOpt) *policyv1alpha1.UpstreamTrafficSetting
}

// UpstreamTrafficSettingGetOpt specifies the options used to filter UpstreamTrafficSetting objects as a part of its getter
type UpstreamTrafficSettingGetOpt struct {
	// MeshService specifies the mesh service known within the cluster
	// Must be specified when retrieving a resource matching the upstream
	// mesh service.
	MeshService *service.MeshService

	// NamespacedName specifies the name and namespace of the resource
	NamespacedName *types.NamespacedName

	// Host specifies the host field of matching UpstreamTrafficSettings
	// This field is not qualified by namespace because, by definition,
	// a properly formatted Host includes a namespace and UpstreamTrafficSetting
	// resources should not target services in different namespaces.
	Host string
}
