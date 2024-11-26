// Package informers centralize informers by creating a single object that
// runs a set of informers, instead of creating different objects
// that each manage their own informer collections.
// A pointer to this object is then shared with all objects that need it.
package informers

import (
	"errors"
	"testing"

	"github.com/flomesh-io/fsm/pkg/version"

	"github.com/rs/zerolog/log"
	smiTrafficAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiAccessInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/informers/externalversions"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSpecInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/informers/externalversions"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiTrafficSplitInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayApiInformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"

	"github.com/flomesh-io/fsm/pkg/constants"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	configInformers "github.com/flomesh-io/fsm/pkg/gen/client/config/informers/externalversions"
	connectorClientset "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned"
	connectorInformers "github.com/flomesh-io/fsm/pkg/gen/client/connector/informers/externalversions"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	machineInformers "github.com/flomesh-io/fsm/pkg/gen/client/machine/informers/externalversions"
	multiclusterClientset "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/clientset/versioned"
	multiclusterInformers "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/informers/externalversions"
	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"
	nsigInformers "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/informers/externalversions"
	networkingClientset "github.com/flomesh-io/fsm/pkg/gen/client/networking/clientset/versioned"
	networkingInformers "github.com/flomesh-io/fsm/pkg/gen/client/networking/informers/externalversions"
	pluginClientset "github.com/flomesh-io/fsm/pkg/gen/client/plugin/clientset/versioned"
	pluginInformers "github.com/flomesh-io/fsm/pkg/gen/client/plugin/informers/externalversions"
	policyClientset "github.com/flomesh-io/fsm/pkg/gen/client/policy/clientset/versioned"
	policyInformers "github.com/flomesh-io/fsm/pkg/gen/client/policy/informers/externalversions"
	xnetworkClientset "github.com/flomesh-io/fsm/pkg/gen/client/xnetwork/clientset/versioned"
	xnetworkInformers "github.com/flomesh-io/fsm/pkg/gen/client/xnetwork/informers/externalversions"

	policyAttachmentClientset "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"
	policyAttachmentInformers "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/informers/externalversions"
)

// InformerCollectionOption is a function that modifies an informer collection
type InformerCollectionOption func(*InformerCollection)

// NewInformerCollection creates a new InformerCollection
func NewInformerCollection(meshName string, stop <-chan struct{}, opts ...InformerCollectionOption) (*InformerCollection, error) {
	ic := &InformerCollection{
		meshName:  meshName,
		informers: map[InformerKey]cache.SharedIndexInformer{},
		//listers:   &Lister{},
	}

	// Execute all of the given options (e.g. set clients, set custom stores, etc.)
	for _, opt := range opts {
		if opt != nil {
			opt(ic)
		}
	}

	if err := ic.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start informer collection")
		return nil, err
	}

	return ic, nil
}

// WithKubeClient sets the kubeClient for the InformerCollection
func WithKubeClient(kubeClient kubernetes.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		// initialize informers
		monitorNamespaceLabel := map[string]string{constants.FSMKubeResourceMonitorAnnotation: ic.meshName}

		labelSelector := fields.SelectorFromSet(monitorNamespaceLabel).String()
		option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.LabelSelector = labelSelector
		})

		nsInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, DefaultKubeEventResyncInterval, option)
		informerFactory := informers.NewSharedInformerFactory(kubeClient, DefaultKubeEventResyncInterval)
		v1api := informerFactory.Core().V1()
		ic.informers[InformerKeyNamespace] = nsInformerFactory.Core().V1().Namespaces().Informer()
		ic.informers[InformerKeyService] = v1api.Services().Informer()
		ic.informers[InformerKeyServiceAccount] = v1api.ServiceAccounts().Informer()
		ic.informers[InformerKeyPod] = v1api.Pods().Informer()
		ic.informers[InformerKeyEndpoints] = v1api.Endpoints().Informer()
		ic.informers[InformerKeyK8sIngressClass] = informerFactory.Networking().V1().IngressClasses().Informer()
		ic.informers[InformerKeyK8sIngress] = informerFactory.Networking().V1().Ingresses().Informer()
		ic.informers[InformerKeySecret] = v1api.Secrets().Informer()
		ic.informers[InformerKeyConfigMap] = v1api.ConfigMaps().Informer()
		ic.informers[InformerKeyNamespaceAll] = v1api.Namespaces().Informer()

		if version.IsEndpointSliceEnabled(kubeClient) {
			ic.informers[InformerKeyEndpointSlices] = informerFactory.Discovery().V1().EndpointSlices().Informer()
		}
	}
}

// WithKubeClientWithoutNamespace sets the kubeClient for the InformerCollection
func WithKubeClientWithoutNamespace(kubeClient kubernetes.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := informers.NewSharedInformerFactory(kubeClient, DefaultKubeEventResyncInterval)
		v1api := informerFactory.Core().V1()
		ic.informers[InformerKeyService] = v1api.Services().Informer()
		ic.informers[InformerKeyServiceAccount] = v1api.ServiceAccounts().Informer()
		ic.informers[InformerKeyPod] = v1api.Pods().Informer()
		ic.informers[InformerKeyEndpoints] = v1api.Endpoints().Informer()
		ic.informers[InformerKeyK8sIngressClass] = informerFactory.Networking().V1().IngressClasses().Informer()
		ic.informers[InformerKeyK8sIngress] = informerFactory.Networking().V1().Ingresses().Informer()
		ic.informers[InformerKeySecret] = v1api.Secrets().Informer()
		ic.informers[InformerKeyConfigMap] = v1api.ConfigMaps().Informer()

		if version.IsEndpointSliceEnabled(kubeClient) {
			ic.informers[InformerKeyEndpointSlices] = informerFactory.Discovery().V1().EndpointSlices().Informer()
		}
	}
}

// WithSMIClients sets the SMI clients for the InformerCollection
func WithSMIClients(smiTrafficSplitClient smiTrafficSplitClient.Interface, smiTrafficSpecClient smiTrafficSpecClient.Interface, smiAccessClient smiTrafficAccessClient.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		accessInformerFactory := smiAccessInformers.NewSharedInformerFactory(smiAccessClient, DefaultKubeEventResyncInterval)
		splitInformerFactory := smiTrafficSplitInformers.NewSharedInformerFactory(smiTrafficSplitClient, DefaultKubeEventResyncInterval)
		specInformerFactory := smiTrafficSpecInformers.NewSharedInformerFactory(smiTrafficSpecClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyTCPRoute] = specInformerFactory.Specs().V1alpha4().TCPRoutes().Informer()
		ic.informers[InformerKeyHTTPRouteGroup] = specInformerFactory.Specs().V1alpha4().HTTPRouteGroups().Informer()
		ic.informers[InformerKeyTrafficTarget] = accessInformerFactory.Access().V1alpha3().TrafficTargets().Informer()
		ic.informers[InformerKeyTrafficSplit] = splitInformerFactory.Split().V1alpha4().TrafficSplits().Informer()
	}
}

// WithConfigClient sets the config client for the InformerCollection
func WithConfigClient(configClient configClientset.Interface, meshConfigName, fsmNamespace string) InformerCollectionOption {
	return func(ic *InformerCollection) {
		listOption := configInformers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, meshConfigName).String()
		})
		meshConfiginformerFactory := configInformers.NewSharedInformerFactoryWithOptions(configClient, DefaultKubeEventResyncInterval, configInformers.WithNamespace(fsmNamespace), listOption)
		mrcInformerFactory := configInformers.NewSharedInformerFactoryWithOptions(configClient, DefaultKubeEventResyncInterval, configInformers.WithNamespace(fsmNamespace))

		ic.informers[InformerKeyMeshConfig] = meshConfiginformerFactory.Config().V1alpha3().MeshConfigs().Informer()
		ic.informers[InformerKeyMeshRootCertificate] = mrcInformerFactory.Config().V1alpha3().MeshRootCertificates().Informer()
	}
}

// WithPolicyClient sets the policy client for the InformerCollection
func WithPolicyClient(policyClient policyClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := policyInformers.NewSharedInformerFactory(policyClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyEgress] = informerFactory.Policy().V1alpha1().Egresses().Informer()
		ic.informers[InformerKeyEgressGateway] = informerFactory.Policy().V1alpha1().EgressGateways().Informer()
		ic.informers[InformerKeyIngressBackend] = informerFactory.Policy().V1alpha1().IngressBackends().Informer()
		ic.informers[InformerKeyUpstreamTrafficSetting] = informerFactory.Policy().V1alpha1().UpstreamTrafficSettings().Informer()
		ic.informers[InformerKeyIsolation] = informerFactory.Policy().V1alpha1().Isolations().Informer()
		ic.informers[InformerKeyRetry] = informerFactory.Policy().V1alpha1().Retries().Informer()
		ic.informers[InformerKeyAccessControl] = informerFactory.Policy().V1alpha1().AccessControls().Informer()
		ic.informers[InformerKeyAccessCert] = informerFactory.Policy().V1alpha1().AccessCerts().Informer()
	}
}

// WithPluginClient sets the plugin client for the InformerCollection
func WithPluginClient(pluginClient pluginClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := pluginInformers.NewSharedInformerFactory(pluginClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyPlugin] = informerFactory.Plugin().V1alpha1().Plugins().Informer()
		ic.informers[InformerKeyPluginChain] = informerFactory.Plugin().V1alpha1().PluginChains().Informer()
		ic.informers[InformerKeyPluginConfig] = informerFactory.Plugin().V1alpha1().PluginConfigs().Informer()
	}
}

// WithMachineClient sets the machine client for the InformerCollection
func WithMachineClient(machineClient machineClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := machineInformers.NewSharedInformerFactory(machineClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyVirtualMachine] = informerFactory.Machine().V1alpha1().VirtualMachines().Informer()
	}
}

// WithConnectorClient sets the connector client for the InformerCollection
func WithConnectorClient(connectorClient connectorClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := connectorInformers.NewSharedInformerFactory(connectorClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyConsulConnector] = informerFactory.Connector().V1alpha1().ConsulConnectors().Informer()
		ic.informers[InformerKeyEurekaConnector] = informerFactory.Connector().V1alpha1().EurekaConnectors().Informer()
		ic.informers[InformerKeyNacosConnector] = informerFactory.Connector().V1alpha1().NacosConnectors().Informer()
		ic.informers[InformerKeyMachineConnector] = informerFactory.Connector().V1alpha1().MachineConnectors().Informer()
		ic.informers[InformerKeyGatewayConnector] = informerFactory.Connector().V1alpha1().GatewayConnectors().Informer()
	}
}

// WithXNetworkClient sets the xnetwork client for the InformerCollection
func WithXNetworkClient(xnetworkClient xnetworkClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := xnetworkInformers.NewSharedInformerFactory(xnetworkClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyXNetworkAccessControl] = informerFactory.Xnetwork().V1alpha1().AccessControls().Informer()
	}
}

// WithMultiClusterClient sets the multicluster client for the InformerCollection
func WithMultiClusterClient(multiclusterClient multiclusterClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := multiclusterInformers.NewSharedInformerFactory(multiclusterClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyServiceExport] = informerFactory.Multicluster().V1alpha1().ServiceExports().Informer()
		ic.informers[InformerKeyServiceImport] = informerFactory.Multicluster().V1alpha1().ServiceImports().Informer()
		ic.informers[InformerKeyGlobalTrafficPolicy] = informerFactory.Multicluster().V1alpha1().GlobalTrafficPolicies().Informer()
	}
}

// WithNetworkingClient sets the networking client for the InformerCollection
func WithNetworkingClient(networkingClient networkingClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := networkingInformers.NewSharedInformerFactory(networkingClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyIngressClass] = informerFactory.Networking().V1().IngressClasses().Informer()
	}
}

// WithIngressClient sets the networking client for the InformerCollection
func WithIngressClient(kubeClient kubernetes.Interface, nsigClient nsigClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := informers.NewSharedInformerFactory(kubeClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyK8sIngressClass] = informerFactory.Networking().V1().IngressClasses().Informer()
		ic.informers[InformerKeyK8sIngress] = informerFactory.Networking().V1().Ingresses().Informer()

		nsigInformerFactory := nsigInformers.NewSharedInformerFactory(nsigClient, DefaultKubeEventResyncInterval)
		ic.informers[InformerKeyNamespacedIngress] = nsigInformerFactory.Networking().V1alpha1().NamespacedIngresses().Informer()
	}
}

// WithPolicyAttachmentClient sets the PolicyAttachment client for the InformerCollection
//func WithPolicyAttachmentClient(policyAttachmentClient policyAttachmentClientset.Interface) InformerCollectionOption {
//	return func(ic *InformerCollection) {
//		informerFactory := policyAttachmentInformers.NewSharedInformerFactory(policyAttachmentClient, DefaultKubeEventResyncInterval)
//
//		ic.informers[InformerKeyRateLimitPolicy] = informerFactory.Gateway().V1alpha1().RateLimitPolicies().Informer()
//		ic.informers[InformerKeySessionStickyPolicy] = informerFactory.Gateway().V1alpha1().SessionStickyPolicies().Informer()
//		ic.informers[InformerKeyLoadBalancerPolicy] = informerFactory.Gateway().V1alpha1().LoadBalancerPolicies().Informer()
//		ic.informers[InformerKeyCircuitBreakingPolicy] = informerFactory.Gateway().V1alpha1().CircuitBreakingPolicies().Informer()
//		ic.informers[InformerKeyAccessControlPolicy] = informerFactory.Gateway().V1alpha1().AccessControlPolicies().Informer()
//		ic.informers[InformerKeyHealthCheckPolicyV1alpha1] = informerFactory.Gateway().V1alpha1().HealthCheckPolicies().Informer()
//		ic.informers[InformerKeyFaultInjectionPolicy] = informerFactory.Gateway().V1alpha1().FaultInjectionPolicies().Informer()
//		ic.informers[InformerKeyUpstreamTLSPolicy] = informerFactory.Gateway().V1alpha1().UpstreamTLSPolicies().Informer()
//		ic.informers[InformerKeyRetryPolicyV1alpha1] = informerFactory.Gateway().V1alpha1().RetryPolicies().Informer()
//	}
//}

// WithPolicyAttachmentClientV2 sets the PolicyAttachment client for the InformerCollection
func WithPolicyAttachmentClientV2(gatewayAPIClient gatewayApiClientset.Interface, policyAttachmentClient policyAttachmentClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		gatewayInformerFactory := gatewayApiInformers.NewSharedInformerFactory(gatewayAPIClient, DefaultKubeEventResyncInterval)
		ic.informers[InformerKeyBackendTLSPolicy] = gatewayInformerFactory.Gateway().V1alpha3().BackendTLSPolicies().Informer()

		informerFactory := policyAttachmentInformers.NewSharedInformerFactory(policyAttachmentClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyBackendLBPolicy] = informerFactory.Gateway().V1alpha2().BackendLBPolicies().Informer()
		ic.informers[InformerKeyHealthCheckPolicyV1alpha2] = informerFactory.Gateway().V1alpha2().HealthCheckPolicies().Informer()
	}
}

// WithGatewayAPIClient sets the gateway api client for the InformerCollection
func WithGatewayAPIClient(gatewayAPIClient gatewayApiClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := gatewayApiInformers.NewSharedInformerFactory(gatewayAPIClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyGatewayAPIGatewayClass] = informerFactory.Gateway().V1().GatewayClasses().Informer()
		ic.informers[InformerKeyGatewayAPIGateway] = informerFactory.Gateway().V1().Gateways().Informer()
		ic.informers[InformerKeyGatewayAPIHTTPRoute] = informerFactory.Gateway().V1().HTTPRoutes().Informer()
		ic.informers[InformerKeyGatewayAPIGRPCRoute] = informerFactory.Gateway().V1().GRPCRoutes().Informer()
		ic.informers[InformerKeyGatewayAPITCPRoute] = informerFactory.Gateway().V1alpha2().TCPRoutes().Informer()
		ic.informers[InformerKeyGatewayAPITLSRoute] = informerFactory.Gateway().V1alpha2().TLSRoutes().Informer()
		ic.informers[InformerKeyGatewayAPIUDPRoute] = informerFactory.Gateway().V1alpha2().UDPRoutes().Informer()
		ic.informers[InformerKeyGatewayAPIReferenceGrant] = informerFactory.Gateway().V1beta1().ReferenceGrants().Informer()
	}
}

func (ic *InformerCollection) run(stop <-chan struct{}) error {
	log.Info().Msg("InformerCollection started")
	var hasSynced []cache.InformerSynced
	var names []string

	if ic.informers == nil {
		return errInitInformers
	}

	for name, informer := range ic.informers {
		if informer == nil {
			continue
		}

		go informer.Run(stop)
		names = append(names, string(name))
		log.Info().Msgf("Waiting for %s informer cache sync...", name)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	log.Info().Msgf("Caches for %v synced successfully", names)

	return nil
}

// Add is only exported for the sake of tests and requires a testing.T to ensure it's
// never used in production. This functionality was added for the express purpose of testing
// flexibility since alternatives can often lead to flaky tests and race conditions
// between the time an object is added to a fake clientset and when that object
// is actually added to the informer `cache.Store`
func (ic *InformerCollection) Add(key InformerKey, obj interface{}, t *testing.T) error {
	if t == nil {
		return errors.New("this method should only be used in tests")
	}

	i, ok := ic.informers[key]
	if !ok {
		t.Errorf("tried to add to nil store with key %s", key)
	}

	return i.GetStore().Add(obj)
}

// Update is only exported for the sake of tests and requires a testing.T to ensure it's
// never used in production. This functionality was added for the express purpose of testing
// flexibility since the alternatives can often lead to flaky tests and race conditions
// between the time an object is added to a fake clientset and when that object
// is actually added to the informer `cache.Store`
func (ic *InformerCollection) Update(key InformerKey, obj interface{}, t *testing.T) error {
	if t == nil {
		return errors.New("this method should only be used in tests")
	}

	i, ok := ic.informers[key]
	if !ok {
		t.Errorf("tried to update to nil store with key %s", key)
	}

	return i.GetStore().Update(obj)
}

// AddEventHandler adds an handler to the informer indexed by the given InformerKey
func (ic *InformerCollection) AddEventHandler(informerKey InformerKey, handler cache.ResourceEventHandler) {
	i, ok := ic.informers[informerKey]
	if !ok {
		log.Info().Msgf("attempted to add event handler for nil informer %s", informerKey)
		return
	}

	_, _ = i.AddEventHandler(handler)
}

// GetByKey retrieves an item (based on the given index) from the store of the informer indexed by the given InformerKey
func (ic *InformerCollection) GetByKey(informerKey InformerKey, objectKey string) (interface{}, bool, error) {
	informer, ok := ic.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil, false, nil
	}

	return informer.GetStore().GetByKey(objectKey)
}

// List returns the contents of the store of the informer indexed by the given InformerKey
func (ic *InformerCollection) List(informerKey InformerKey) []interface{} {
	informer, ok := ic.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil
	}

	return informer.GetStore().List()
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (ic *InformerCollection) IsMonitoredNamespace(namespace string) bool {
	_, exists, _ := ic.informers[InformerKeyNamespace].GetStore().GetByKey(namespace)
	return exists
}

// GetListers returns the listers for the informers
//func (ic *InformerCollection) GetListers() *Lister {
//	return ic.listers
//}

//gocyclo:ignore
//func (ic *InformerCollection) GetGatewayResourcesFromCache(resourceType ResourceType, shouldSort bool) []client.Object {
//	resources := make([]client.Object, 0)
//
//	switch resourceType {
//	case GatewayClassesResourceType:
//		classes, err := ic.listers.GatewayClass.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get GatewayClasses: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(classes, constants.GatewayClassGVK)
//	case GatewaysResourceType:
//		gateways, err := ic.listers.Gateway.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get Gateways: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(gateways, constants.GatewayGVK)
//	case HTTPRoutesResourceType:
//		routes, err := ic.listers.HTTPRoute.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get HTTPRoutes: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(routes, constants.HTTPRouteGVK)
//	case GRPCRoutesResourceType:
//		routes, err := ic.listers.GRPCRoute.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get GRPCRouts: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(routes, constants.GRPCRouteGVK)
//	case TLSRoutesResourceType:
//		routes, err := ic.listers.TLSRoute.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get TLSRoutes: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(routes, constants.TLSRouteGVK)
//	case TCPRoutesResourceType:
//		routes, err := ic.listers.TCPRoute.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get TCPRoutes: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(routes, constants.TCPRouteGVK)
//	case UDPRoutesResourceType:
//		routes, err := ic.listers.UDPRoute.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get UDPRoutes: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(routes, constants.UDPRouteGVK)
//	case ReferenceGrantResourceType:
//		grants, err := ic.listers.ReferenceGrant.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get ReferenceGrants: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(grants, constants.ReferenceGrantGVK)
//	case UpstreamTLSPoliciesResourceType:
//		policies, err := ic.listers.UpstreamTLSPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get UpstreamTLSPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.UpstreamTLSPolicyGVK)
//	case RateLimitPoliciesResourceType:
//		policies, err := ic.listers.RateLimitPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get RateLimitPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.RateLimitPolicyGVK)
//	case AccessControlPoliciesResourceType:
//		policies, err := ic.listers.AccessControlPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get AccessControlPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.AccessControlPolicyGVK)
//	case FaultInjectionPoliciesResourceType:
//		policies, err := ic.listers.FaultInjectionPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get FaultInjectionPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.FaultInjectionPolicyGVK)
//	case SessionStickyPoliciesResourceType:
//		policies, err := ic.listers.SessionStickyPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get SessionStickyPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.SessionStickyPolicyGVK)
//	case LoadBalancerPoliciesResourceType:
//		policies, err := ic.listers.LoadBalancerPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get LoadBalancerPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.LoadBalancerPolicyGVK)
//	case CircuitBreakingPoliciesResourceType:
//		policies, err := ic.listers.CircuitBreakingPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get CircuitBreakingPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.CircuitBreakingPolicyGVK)
//	case HealthCheckPoliciesResourceType:
//		policies, err := ic.listers.HealthCheckPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get HealthCheckPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.HealthCheckPolicyGVK)
//	case RetryPoliciesResourceType:
//		policies, err := ic.listers.RetryPolicy.List(selectAll)
//		if err != nil {
//			log.Error().Msgf("Failed to get RetryPolicies: %s", err)
//			return resources
//		}
//		resources = setGroupVersionKind(policies, constants.RetryPolicyGVK)
//	default:
//		log.Error().Msgf("Unknown resource type: %s", resourceType)
//		return nil
//	}
//
//	if shouldSort {
//		sort.Slice(resources, func(i, j int) bool {
//			if resources[i].GetCreationTimestamp().Time.Equal(resources[j].GetCreationTimestamp().Time) {
//				return client.ObjectKeyFromObject(resources[i]).String() < client.ObjectKeyFromObject(resources[j]).String()
//			}
//
//			return resources[i].GetCreationTimestamp().Time.Before(resources[j].GetCreationTimestamp().Time)
//		})
//	}
//
//	return resources
//}
//
//func setGroupVersionKind[T GatewayAPIResource](objects []T, gvk schema.GroupVersionKind) []client.Object {
//	resources := make([]client.Object, 0)
//
//	for _, obj := range objects {
//		obj := client.Object(obj)
//		obj.GetObjectKind().SetGroupVersionKind(gvk)
//		resources = append(resources, obj)
//	}
//
//	return resources
//}
