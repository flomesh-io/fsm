package gateway

import (
	"context"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8scache "k8s.io/client-go/tools/cache"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/flomesh-io/fsm/pkg/announcements"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/gateway/cache"
	"github.com/flomesh-io/fsm/pkg/gateway/repo"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

var (
	log = logger.New("controller-gatewayapi")
)

// NewGatewayAPIController returns a gateway.Controller interface related to functionality provided by the resources in the plugin.flomesh.io API group
func NewGatewayAPIController(informerCollection *fsminformers.InformerCollection, kubeClient kubernetes.Interface, gatewayAPIClient gatewayApiClientset.Interface, msgBroker *messaging.Broker, cfg configurator.Configurator, meshName, fsmVersion string) Controller {
	return newClient(informerCollection, kubeClient, gatewayAPIClient, msgBroker, cfg, meshName, fsmVersion)
}

func newClient(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, gatewayAPIClient gatewayApiClientset.Interface, msgBroker *messaging.Broker, cfg configurator.Configurator, meshName, fsmVersion string) *client {
	fsmGatewayClass := &gwv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.FSMGatewayClassName,
			Labels: map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.FSMAppInstanceLabelKey: meshName,
				constants.FSMAppVersionLabelKey:  fsmVersion,
				constants.AppLabel:               constants.FSMGatewayName,
			},
		},
		Spec: gwv1.GatewayClassSpec{
			ControllerName: constants.GatewayController,
		},
	}

	if _, err := gatewayAPIClient.GatewayV1().
		GatewayClasses().
		Create(context.TODO(), fsmGatewayClass, metav1.CreateOptions{}); err != nil {
		log.Warn().Msgf("Failed to create FSM GatewayClass: %s", err)
	}

	c := &client{
		informers:  informerCollection,
		kubeClient: kubeClient,
		msgBroker:  msgBroker,
		cfg:        cfg,
		cache:      cache.NewGatewayCache(informerCollection, kubeClient, cfg),
	}

	// Initialize informers
	for _, informerKey := range []fsminformers.InformerKey{
		fsminformers.InformerKeyService,
		fsminformers.InformerKeyServiceImport,
		fsminformers.InformerKeyEndpoints,
		fsminformers.InformerKeyEndpointSlices,
		fsminformers.InformerKeySecret,
		fsminformers.InformerKeyConfigMap,
		fsminformers.InformerKeyGatewayAPIGatewayClass,
		fsminformers.InformerKeyGatewayAPIGateway,
		fsminformers.InformerKeyGatewayAPIHTTPRoute,
		fsminformers.InformerKeyGatewayAPIGRPCRoute,
		fsminformers.InformerKeyGatewayAPITLSRoute,
		fsminformers.InformerKeyGatewayAPITCPRoute,
		fsminformers.InformerKeyGatewayAPIUDPRoute,
		fsminformers.InformerKeyGatewayAPIReferenceGrant,
		fsminformers.InformerKeyRateLimitPolicy,
		fsminformers.InformerKeySessionStickyPolicy,
		fsminformers.InformerKeyLoadBalancerPolicy,
		fsminformers.InformerKeyCircuitBreakingPolicy,
		fsminformers.InformerKeyAccessControlPolicy,
		fsminformers.InformerKeyHealthCheckPolicy,
		fsminformers.InformerKeyFaultInjectionPolicy,
		fsminformers.InformerKeyUpstreamTLSPolicy,
		fsminformers.InformerKeyRetryPolicy,
	} {
		if eventTypes := getEventTypesByInformerKey(informerKey); eventTypes != nil {
			c.informers.AddEventHandler(informerKey, c.getEventHandlerFuncs(eventTypes))
		}
	}

	return c
}

func (c *client) getEventHandlerFuncs(eventTypes *k8s.EventTypes) k8scache.ResourceEventHandlerFuncs {
	return k8scache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAddFunc(eventTypes),
		UpdateFunc: c.onUpdateFunc(eventTypes),
		DeleteFunc: c.onDeleteFunc(eventTypes),
	}
}

func (c *client) onAddFunc(eventTypes *k8s.EventTypes) func(obj interface{}) {
	return func(obj interface{}) {
		if !c.shouldObserve(nil, obj) {
			return
		}
		logResourceEvent(log, eventTypes.Add, obj)
		c.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
			Kind:   eventTypes.Add,
			NewObj: obj,
			OldObj: nil,
		})
	}
}

func (c *client) onUpdateFunc(eventTypes *k8s.EventTypes) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		if !c.shouldObserve(oldObj, newObj) {
			return
		}
		logResourceEvent(log, eventTypes.Update, newObj)
		c.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
			Kind:   eventTypes.Update,
			NewObj: newObj,
			OldObj: oldObj,
		})
	}
}

func (c *client) onDeleteFunc(eventTypes *k8s.EventTypes) func(obj interface{}) {
	return func(obj interface{}) {
		if !c.shouldObserve(obj, nil) {
			return
		}
		logResourceEvent(log, eventTypes.Delete, obj)
		c.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
			Kind:   eventTypes.Delete,
			NewObj: nil,
			OldObj: obj,
		})
	}
}

func (c *client) shouldObserve(oldObj, newObj interface{}) bool {
	return c.onChange(oldObj, newObj)
}

func (c *client) onChange(oldObj, newObj interface{}) bool {
	if newObj == nil {
		return c.cache.Delete(oldObj)
	}

	if oldObj == nil {
		return c.cache.Insert(newObj)
	}

	if cmp.Equal(oldObj, newObj) {
		return false
	}

	del := c.cache.Delete(oldObj)
	ins := c.cache.Insert(newObj)

	return del || ins
}

func (c *client) OnAdd(obj interface{}, isInitialList bool) {
	if eventTypes := getEventTypesByObjectType(obj); eventTypes != nil {
		c.onAddFunc(eventTypes)(obj)
	}
}

func (c *client) OnUpdate(oldObj, newObj interface{}) {
	if eventTypes := getEventTypesByObjectType(newObj); eventTypes != nil {
		c.onUpdateFunc(eventTypes)(oldObj, newObj)
	}
}

func (c *client) OnDelete(obj interface{}) {
	if eventTypes := getEventTypesByObjectType(obj); eventTypes != nil {
		c.onDeleteFunc(eventTypes)(obj)
	}
}

func getEventTypesByObjectType(obj interface{}) *k8s.EventTypes {
	switch obj.(type) {
	case *corev1.Service:
		return getEventTypesByInformerKey(fsminformers.InformerKeyService)
	case *mcsv1alpha1.ServiceImport:
		return getEventTypesByInformerKey(fsminformers.InformerKeyServiceImport)
	case *corev1.Endpoints:
		return getEventTypesByInformerKey(fsminformers.InformerKeyEndpoints)
	case *discoveryv1.EndpointSlice:
		return getEventTypesByInformerKey(fsminformers.InformerKeyEndpointSlices)
	case *corev1.Secret:
		return getEventTypesByInformerKey(fsminformers.InformerKeySecret)
	case *corev1.ConfigMap:
		return getEventTypesByInformerKey(fsminformers.InformerKeyConfigMap)
	case *gwv1.GatewayClass:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIGatewayClass)
	case *gwv1.Gateway:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIGateway)
	case *gwv1.HTTPRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIHTTPRoute)
	case *gwv1.GRPCRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIGRPCRoute)
	case *gwv1alpha2.TLSRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPITLSRoute)
	case *gwv1alpha2.TCPRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPITCPRoute)
	case *gwv1alpha2.UDPRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIUDPRoute)
	case *gwv1beta1.ReferenceGrant:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIReferenceGrant)
	case *gwpav1alpha1.RateLimitPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyRateLimitPolicy)
	case *gwpav1alpha1.SessionStickyPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeySessionStickyPolicy)
	case *gwpav1alpha1.LoadBalancerPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyLoadBalancerPolicy)
	case *gwpav1alpha1.CircuitBreakingPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyCircuitBreakingPolicy)
	case *gwpav1alpha1.AccessControlPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyAccessControlPolicy)
	case *gwpav1alpha1.HealthCheckPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyHealthCheckPolicy)
	case *gwpav1alpha1.FaultInjectionPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyFaultInjectionPolicy)
	case *gwpav1alpha1.UpstreamTLSPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyUpstreamTLSPolicy)
	case *gwpav1alpha1.RetryPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyRetryPolicy)
	}

	return nil
}

func getEventTypesByInformerKey(informerKey fsminformers.InformerKey) *k8s.EventTypes {
	switch informerKey {
	case fsminformers.InformerKeyService:
		return &k8s.EventTypes{
			Add:    announcements.ServiceAdded,
			Update: announcements.ServiceUpdated,
			Delete: announcements.ServiceDeleted,
		}
	case fsminformers.InformerKeyServiceImport:
		return &k8s.EventTypes{
			Add:    announcements.ServiceImportAdded,
			Update: announcements.ServiceImportUpdated,
			Delete: announcements.ServiceImportDeleted,
		}
	case fsminformers.InformerKeyEndpointSlices:
		return &k8s.EventTypes{
			Add:    announcements.EndpointSlicesAdded,
			Update: announcements.EndpointSlicesUpdated,
			Delete: announcements.EndpointSlicesDeleted,
		}
	case fsminformers.InformerKeyEndpoints:
		return &k8s.EventTypes{
			Add:    announcements.EndpointAdded,
			Update: announcements.EndpointUpdated,
			Delete: announcements.EndpointDeleted,
		}
	case fsminformers.InformerKeySecret:
		return &k8s.EventTypes{
			Add:    announcements.SecretAdded,
			Update: announcements.SecretUpdated,
			Delete: announcements.SecretDeleted,
		}
	case fsminformers.InformerKeyConfigMap:
		return &k8s.EventTypes{
			Add:    announcements.ConfigMapAdded,
			Update: announcements.ConfigMapUpdated,
			Delete: announcements.ConfigMapDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIGatewayClass:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIGatewayClassAdded,
			Update: announcements.GatewayAPIGatewayClassUpdated,
			Delete: announcements.GatewayAPIGatewayClassDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIGateway:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIGatewayAdded,
			Update: announcements.GatewayAPIGatewayUpdated,
			Delete: announcements.GatewayAPIGatewayDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIHTTPRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIHTTPRouteAdded,
			Update: announcements.GatewayAPIHTTPRouteUpdated,
			Delete: announcements.GatewayAPIHTTPRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIGRPCRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIGRPCRouteAdded,
			Update: announcements.GatewayAPIGRPCRouteUpdated,
			Delete: announcements.GatewayAPIGRPCRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPITLSRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPITLSRouteAdded,
			Update: announcements.GatewayAPITLSRouteUpdated,
			Delete: announcements.GatewayAPITLSRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPITCPRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPITCPRouteAdded,
			Update: announcements.GatewayAPITCPRouteUpdated,
			Delete: announcements.GatewayAPITCPRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIUDPRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIUDPRouteAdded,
			Update: announcements.GatewayAPIUDPRouteUpdated,
			Delete: announcements.GatewayAPIUDPRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIReferenceGrant:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIReferenceGrantAdded,
			Update: announcements.GatewayAPIReferenceGrantUpdated,
			Delete: announcements.GatewayAPIReferenceGrantDeleted,
		}
	case fsminformers.InformerKeyRateLimitPolicy:
		return &k8s.EventTypes{
			Add:    announcements.RateLimitPolicyAdded,
			Update: announcements.RateLimitPolicyUpdated,
			Delete: announcements.RateLimitPolicyDeleted,
		}
	case fsminformers.InformerKeySessionStickyPolicy:
		return &k8s.EventTypes{
			Add:    announcements.SessionStickyPolicyAdded,
			Update: announcements.SessionStickyPolicyUpdated,
			Delete: announcements.SessionStickyPolicyDeleted,
		}
	case fsminformers.InformerKeyLoadBalancerPolicy:
		return &k8s.EventTypes{
			Add:    announcements.LoadBalancerPolicyAdded,
			Update: announcements.LoadBalancerPolicyUpdated,
			Delete: announcements.LoadBalancerPolicyDeleted,
		}
	case fsminformers.InformerKeyCircuitBreakingPolicy:
		return &k8s.EventTypes{
			Add:    announcements.CircuitBreakingPolicyAdded,
			Update: announcements.CircuitBreakingPolicyUpdated,
			Delete: announcements.CircuitBreakingPolicyDeleted,
		}
	case fsminformers.InformerKeyAccessControlPolicy:
		return &k8s.EventTypes{
			Add:    announcements.AccessControlPolicyAdded,
			Update: announcements.AccessControlPolicyUpdated,
			Delete: announcements.AccessControlPolicyDeleted,
		}
	case fsminformers.InformerKeyHealthCheckPolicy:
		return &k8s.EventTypes{
			Add:    announcements.HealthCheckPolicyAdded,
			Update: announcements.HealthCheckPolicyUpdated,
			Delete: announcements.HealthCheckPolicyDeleted,
		}
	case fsminformers.InformerKeyFaultInjectionPolicy:
		return &k8s.EventTypes{
			Add:    announcements.FaultInjectionPolicyAdded,
			Update: announcements.FaultInjectionPolicyUpdated,
			Delete: announcements.FaultInjectionPolicyDeleted,
		}
	case fsminformers.InformerKeyUpstreamTLSPolicy:
		return &k8s.EventTypes{
			Add:    announcements.UpstreamTLSPolicyAdded,
			Update: announcements.UpstreamTLSPolicyUpdated,
			Delete: announcements.UpstreamTLSPolicyDeleted,
		}
	case fsminformers.InformerKeyRetryPolicy:
		return &k8s.EventTypes{
			Add:    announcements.RetryPolicyAttachmentAdded,
			Update: announcements.RetryPolicyAttachmentUpdated,
			Delete: announcements.RetryPolicyAttachmentDeleted,
		}
	}

	return nil
}

// NeedLeaderElection implements the LeaderElectionRunnable interface
// to indicate that this should be started without requiring the leader lock.
// The reason is it writes to the local repo which is in the same pod.
func (c *client) NeedLeaderElection() bool {
	return false
}

// Start starts the backend broadcast listener
func (c *client) Start(_ context.Context) error {
	// Start broadcast listener thread
	s := repo.NewServer(c.cfg, c.msgBroker, c.cache)
	go s.BroadcastListener()

	return nil
}

func logResourceEvent(parent zerolog.Logger, event announcements.Kind, obj interface{}) {
	log := parent.With().Str("event", event.String()).Logger()
	o, err := meta.Accessor(obj)
	if err != nil {
		log.Error().Err(err).Msg("error parsing object, ignoring")
		return
	}
	name := o.GetName()
	if o.GetNamespace() != "" {
		name = o.GetNamespace() + "/" + name
	}
	log.Debug().Str("resource_name", name).Msg("received kubernetes resource event")
}
