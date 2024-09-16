package policy

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	policyV1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/service"
)

const (
	// kindSvcAccount is the ServiceAccount kind
	kindSvcAccount = "ServiceAccount"
)

// NewPolicyController returns a policy.Controller interface related to functionality provided by the resources in the policy.flomesh.io API group
func NewPolicyController(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, kubeController k8s.Controller, msgBroker *messaging.Broker) *Client {
	client := &Client{
		informers:      informerCollection,
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return kubeController.IsMonitoredNamespace(object.GetNamespace())
	}

	egressEventTypes := k8s.EventTypes{
		Add:    announcements.EgressAdded,
		Update: announcements.EgressUpdated,
		Delete: announcements.EgressDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyEgress, k8s.GetEventHandlerFuncs(shouldObserve, egressEventTypes, msgBroker))

	egressGatewayEventTypes := k8s.EventTypes{
		Add:    announcements.EgressGatewayAdded,
		Update: announcements.EgressGatewayUpdated,
		Delete: announcements.EgressGatewayDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyEgressGateway, k8s.GetEventHandlerFuncs(shouldObserve, egressGatewayEventTypes, msgBroker))

	ingressBackendEventTypes := k8s.EventTypes{
		Add:    announcements.IngressBackendAdded,
		Update: announcements.IngressBackendUpdated,
		Delete: announcements.IngressBackendDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyIngressBackend, k8s.GetEventHandlerFuncs(shouldObserve, ingressBackendEventTypes, msgBroker))

	aclEventTypes := k8s.EventTypes{
		Add:    announcements.AccessControlAdded,
		Update: announcements.AccessControlUpdated,
		Delete: announcements.AccessControlDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyAccessControl, k8s.GetEventHandlerFuncs(shouldObserve, aclEventTypes, msgBroker))

	acertEventTypes := k8s.EventTypes{
		Add:    announcements.AccessCertAdded,
		Update: announcements.AccessCertUpdated,
		Delete: announcements.AccessCertDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyAccessCert, k8s.GetEventHandlerFuncs(shouldObserve, acertEventTypes, msgBroker))

	isolationEventTypes := k8s.EventTypes{
		Add:    announcements.IsolationPolicyAdded,
		Update: announcements.IsolationPolicyUpdated,
		Delete: announcements.IsolationPolicyDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyIsolation, k8s.GetEventHandlerFuncs(nil, isolationEventTypes, msgBroker))

	retryEventTypes := k8s.EventTypes{
		Add:    announcements.RetryPolicyAdded,
		Update: announcements.RetryPolicyUpdated,
		Delete: announcements.RetryPolicyDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyRetry, k8s.GetEventHandlerFuncs(shouldObserve, retryEventTypes, msgBroker))

	upstreamTrafficSettingEventTypes := k8s.EventTypes{
		Add:    announcements.UpstreamTrafficSettingAdded,
		Update: announcements.UpstreamTrafficSettingUpdated,
		Delete: announcements.UpstreamTrafficSettingDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyUpstreamTrafficSetting, k8s.GetEventHandlerFuncs(shouldObserve, upstreamTrafficSettingEventTypes, msgBroker))

	return client
}

// ListIsolationPolicies returns the Isolation policies
func (c *Client) ListIsolationPolicies() []*policyV1alpha1.Isolation {
	var isolations []*policyV1alpha1.Isolation
	for _, isolationIface := range c.informers.List(informers.InformerKeyIsolation) {
		isolation := isolationIface.(*policyV1alpha1.Isolation)
		isolations = append(isolations, isolation)
	}

	return isolations
}

// ListEgressGateways lists egress gateways
func (c *Client) ListEgressGateways() []*policyV1alpha1.EgressGateway {
	var egressGateways []*policyV1alpha1.EgressGateway
	for _, egressGatewayIface := range c.informers.List(informers.InformerKeyEgressGateway) {
		egressGateway := egressGatewayIface.(*policyV1alpha1.EgressGateway)
		egressGateways = append(egressGateways, egressGateway)
	}

	return egressGateways
}

// ListEgressPoliciesForSourceIdentity lists the Egress policies for the given source identity based on service accounts
func (c *Client) ListEgressPoliciesForSourceIdentity(source identity.K8sServiceAccount) []*policyV1alpha1.Egress {
	var policies []*policyV1alpha1.Egress

	for _, egressIface := range c.informers.List(informers.InformerKeyEgress) {
		egressPolicy := egressIface.(*policyV1alpha1.Egress)

		if !c.kubeController.IsMonitoredNamespace(egressPolicy.Namespace) {
			continue
		}

		for _, sourceSpec := range egressPolicy.Spec.Sources {
			if sourceSpec.Kind == kindSvcAccount && sourceSpec.Name == source.Name && sourceSpec.Namespace == source.Namespace {
				policies = append(policies, egressPolicy)
			}
		}
	}

	return policies
}

// GetEgressSourceSecret returns the secret resource that matches the given options
func (c *Client) GetEgressSourceSecret(secretReference corev1.SecretReference) (*corev1.Secret, error) {
	return c.kubeClient.CoreV1().Secrets(secretReference.Namespace).
		Get(context.Background(), secretReference.Name, metav1.GetOptions{})
}

// GetIngressBackendPolicy returns the IngressBackend policy for the given backend MeshService
func (c *Client) GetIngressBackendPolicy(svc service.MeshService) *policyV1alpha1.IngressBackend {
	for _, ingressBackendIface := range c.informers.List(informers.InformerKeyIngressBackend) {
		ingressBackend := ingressBackendIface.(*policyV1alpha1.IngressBackend)

		if ingressBackend.Namespace != svc.Namespace {
			continue
		}

		// Return the first IngressBackend corresponding to the given MeshService.
		// Multiple IngressBackend policies for the same backend will be prevented
		// using a validating webhook.
		for _, backend := range ingressBackend.Spec.Backends {
			// we need to check ports to allow ingress to multiple ports on the same svc
			if backend.Name == svc.Name && backend.Port.Number == int(svc.TargetPort) {
				return ingressBackend
			}
		}
	}

	return nil
}

// ListRetryPolicies returns the retry policies for the given source identity based on service accounts.
func (c *Client) ListRetryPolicies(source identity.K8sServiceAccount) []*policyV1alpha1.Retry {
	var retries []*policyV1alpha1.Retry

	for _, retryInterface := range c.informers.List(informers.InformerKeyRetry) {
		retry := retryInterface.(*policyV1alpha1.Retry)
		if retry.Spec.Source.Kind == kindSvcAccount && retry.Spec.Source.Name == source.Name && retry.Spec.Source.Namespace == source.Namespace {
			retries = append(retries, retry)
		}
	}

	return retries
}

// GetAccessControlPolicy returns the AccessControl policy for the given backend MeshService
func (c *Client) GetAccessControlPolicy(svc service.MeshService) *policyV1alpha1.AccessControl {
	aclIfaces := c.informers.List(informers.InformerKeyAccessControl)
	for _, aclIface := range aclIfaces {
		acl := aclIface.(*policyV1alpha1.AccessControl)

		if acl.Namespace != svc.Namespace {
			continue
		}

		// Return the first AccessControlBackend corresponding to the given MeshService.
		// Multiple AccessControlBackend policies for the same backend will be prevented
		// using a validating webhook.
		for _, backend := range acl.Spec.Backends {
			// we need to check ports to allow ingress to multiple ports on the same svc
			if backend.Name == svc.Name && backend.Port.Number == int(svc.TargetPort) {
				return acl
			}
		}
	}
	for _, aclIface := range aclIfaces {
		acl := aclIface.(*policyV1alpha1.AccessControl)
		if len(acl.Spec.Backends) == 0 {
			return acl
		}
	}
	return nil
}

// GetUpstreamTrafficSetting returns the UpstreamTrafficSetting resource that matches the given options
func (c *Client) GetUpstreamTrafficSetting(options UpstreamTrafficSettingGetOpt) *policyV1alpha1.UpstreamTrafficSetting {
	if options.MeshService == nil && options.NamespacedName == nil && options.Host == "" {
		log.Error().Msgf("No option specified to get UpstreamTrafficSetting resource")
		return nil
	}

	if options.NamespacedName != nil {
		// Filter by namespaced name
		resource, exists, err := c.informers.GetByKey(informers.InformerKeyUpstreamTrafficSetting, options.NamespacedName.String())
		if exists && err == nil {
			return resource.(*policyV1alpha1.UpstreamTrafficSetting)
		}
		return nil
	}

	// Filter by MeshService
	for _, resource := range c.informers.List(informers.InformerKeyUpstreamTrafficSetting) {
		upstreamTrafficSetting := resource.(*policyV1alpha1.UpstreamTrafficSetting)

		if upstreamTrafficSetting.Spec.Host == options.Host {
			return upstreamTrafficSetting
		}

		if upstreamTrafficSetting.Namespace == options.MeshService.Namespace &&
			(upstreamTrafficSetting.Spec.Host == options.MeshService.PolicyName(true) ||
				upstreamTrafficSetting.Spec.Host == options.MeshService.PolicyName(false) ||
				upstreamTrafficSetting.Spec.Host == options.MeshService.FQDN()) {
			return upstreamTrafficSetting
		}
	}

	return nil
}
