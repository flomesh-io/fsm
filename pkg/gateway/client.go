package gateway

import (
	"context"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/flomesh-io/fsm/pkg/announcements"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/version"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scache "k8s.io/client-go/tools/cache"
	crClient "sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/constants"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	cctx "github.com/flomesh-io/fsm/pkg/context"
	gwprocessorv2 "github.com/flomesh-io/fsm/pkg/gateway/processor/v2"
	"github.com/flomesh-io/fsm/pkg/gateway/repo"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// NewGatewayAPIController returns a gateway.Controller interface related to functionality provided by the resources in the gateway.flomesh.io API group
func NewGatewayAPIController(ctx *cctx.ControllerContext) gwtypes.Controller {
	return newClient(ctx)
}

func newClient(ctx *cctx.ControllerContext) *client {
	gatewayAPIClient, err := gatewayApiClientset.NewForConfig(ctx.KubeConfig)
	if err != nil {
		panic(err)
	}

	fsmGatewayClass := &gwv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.FSMGatewayClassName,
			Labels: map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.FSMAppInstanceLabelKey: ctx.MeshName,
				constants.FSMAppVersionLabelKey:  ctx.FSMVersion,
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
		msgBroker: ctx.MsgBroker,
		cfg:       ctx.Configurator,
		processor: gwprocessorv2.NewGatewayProcessor(ctx),
	}

	// Initialize informers
	informers := map[fsminformers.InformerKey]crClient.Object{
		fsminformers.InformerKeyService:                   &corev1.Service{},
		fsminformers.InformerKeyServiceImport:             &mcsv1alpha1.ServiceImport{},
		fsminformers.InformerKeyEndpoints:                 &corev1.Endpoints{},
		fsminformers.InformerKeySecret:                    &corev1.Secret{},
		fsminformers.InformerKeyConfigMap:                 &corev1.ConfigMap{},
		fsminformers.InformerKeyGatewayAPIGatewayClass:    &gwv1.GatewayClass{},
		fsminformers.InformerKeyGatewayAPIGateway:         &gwv1.Gateway{},
		fsminformers.InformerKeyGatewayAPIHTTPRoute:       &gwv1.HTTPRoute{},
		fsminformers.InformerKeyGatewayAPIGRPCRoute:       &gwv1.GRPCRoute{},
		fsminformers.InformerKeyGatewayAPITLSRoute:        &gwv1alpha2.TLSRoute{},
		fsminformers.InformerKeyGatewayAPITCPRoute:        &gwv1alpha2.TCPRoute{},
		fsminformers.InformerKeyGatewayAPIUDPRoute:        &gwv1alpha2.UDPRoute{},
		fsminformers.InformerKeyGatewayAPIReferenceGrant:  &gwv1beta1.ReferenceGrant{},
		fsminformers.InformerKeyHealthCheckPolicyV1alpha2: &gwpav1alpha2.HealthCheckPolicy{},
		fsminformers.InformerKeyBackendTLSPolicy:          &gwv1alpha3.BackendTLSPolicy{},
		fsminformers.InformerKeyBackendLBPolicy:           &gwpav1alpha2.BackendLBPolicy{},
		fsminformers.InformerKeyRouteRuleFilterPolicy:     &gwpav1alpha2.RouteRuleFilterPolicy{},
		fsminformers.InformerKeyNamespace:                 &corev1.Namespace{},
		fsminformers.InformerKeyFilter:                    &extv1alpha1.Filter{},
		fsminformers.InformerKeyFilterDefinition:          &extv1alpha1.FilterDefinition{},
		fsminformers.InformerKeyListenerFilter:            &extv1alpha1.ListenerFilter{},
		fsminformers.InformerKeyFilterConfig:              &extv1alpha1.FilterConfig{},
		fsminformers.InformerKeyCircuitBreaker:            &extv1alpha1.CircuitBreaker{},
		fsminformers.InformerKeyFaultInjection:            &extv1alpha1.FaultInjection{},
		fsminformers.InformerKeyRateLimit:                 &extv1alpha1.RateLimit{},
		fsminformers.InformerKeyGatewayHTTPLog:            &extv1alpha1.HTTPLog{},
		fsminformers.InformerKeyGatewayMetrics:            &extv1alpha1.Metrics{},
		fsminformers.InformerKeyGatewayZipkin:             &extv1alpha1.Zipkin{},
		fsminformers.InformerKeyGatewayProxyTag:           &extv1alpha1.ProxyTag{},
		fsminformers.InformerKeyGatewayExternalRateLimit:  &extv1alpha1.ExternalRateLimit{},
		fsminformers.InformerKeyGatewayIPRestriction:      &extv1alpha1.IPRestriction{},
		fsminformers.InformerKeyGatewayRequestTermination: &extv1alpha1.RequestTermination{},
		fsminformers.InformerKeyGatewayConcurrencyLimit:   &extv1alpha1.ConcurrencyLimit{},
		fsminformers.InformerKeyGatewayDNSModifier:        &extv1alpha1.DNSModifier{},
	}

	if version.IsEndpointSliceEnabled(ctx.KubeClient) {
		informers[fsminformers.InformerKeyEndpointSlices] = &discoveryv1.EndpointSlice{}
	}

	for informerKey, resource := range informers {
		if eventTypes := getEventTypesByInformerKey(informerKey); eventTypes != nil {
			c.informOnResource(ctx, resource, c.getEventHandlerFuncs(eventTypes))
		}
	}

	return c
}

func (c *client) informOnResource(ctx *cctx.ControllerContext, obj crClient.Object, handler k8scache.ResourceEventHandlerFuncs) {
	ch := ctx.Manager.GetCache()

	informer, err := ch.GetInformer(context.Background(), obj)
	if err != nil {
		panic(err)
	}

	_, err = informer.AddEventHandler(handler)
	if err != nil {
		panic(err)
	}
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
		return c.processor.Delete(oldObj)
	}

	if oldObj == nil {
		return c.processor.Insert(newObj)
	}

	if cmp.Equal(oldObj, newObj) {
		return false
	}

	del := c.processor.Delete(oldObj)
	ins := c.processor.Insert(newObj)

	return del || ins
}

func (c *client) OnAdd(obj interface{}, _ bool) {
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

// NeedLeaderElection implements the LeaderElectionRunnable interface
// to indicate that this should be started without requiring the leader lock.
// The reason is it writes to the local repo which is in the same pod.
func (c *client) NeedLeaderElection() bool {
	return false
}

// Start starts the backend broadcast listener
func (c *client) Start(_ context.Context) error {
	// Start broadcast listener thread
	s := repo.NewServer(c.cfg, c.msgBroker, c.processor)
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
