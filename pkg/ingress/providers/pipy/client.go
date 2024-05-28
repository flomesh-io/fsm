package pipy

import (
	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	k8scache "k8s.io/client-go/tools/cache"
	crClient "sigs.k8s.io/controller-runtime/pkg/client"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	cctx "github.com/flomesh-io/fsm/pkg/context"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/cache"
	"github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/repo"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("controller-gatewayapi")
)

// NewIngressController returns an ingress.Controller interface related to functionality provided by the resources in the k8s ingress API group
func NewIngressController(ctx *cctx.ControllerContext) Controller {
	return newClient(ctx)
}

func newClient(ctx *cctx.ControllerContext) *client {
	c := &client{
		msgBroker: ctx.MsgBroker,
		cfg:       ctx.Configurator,
		cache:     cache.NewCache(ctx),
	}

	// Initialize informers
	for informerKey, obj := range map[fsminformers.InformerKey]crClient.Object{
		fsminformers.InformerKeyService:         &corev1.Service{},
		fsminformers.InformerKeyServiceImport:   &mcsv1alpha1.ServiceImport{},
		fsminformers.InformerKeyEndpoints:       &corev1.Endpoints{},
		fsminformers.InformerKeySecret:          &corev1.Secret{},
		fsminformers.InformerKeyK8sIngressClass: &networkingv1.IngressClass{},
		fsminformers.InformerKeyK8sIngress:      &networkingv1.Ingress{},
	} {
		if eventTypes := getEventTypesByInformerKey(informerKey); eventTypes != nil {
			c.informOnResource(ctx, obj, c.getEventHandlerFuncs(eventTypes))
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
		return c.cache.OnDelete(oldObj)
	}

	if oldObj == nil {
		return c.cache.OnAdd(newObj)
	}

	if cmp.Equal(oldObj, newObj) {
		return false
	}

	return c.cache.OnUpdate(oldObj, newObj)
}

//func getEventTypesByObjectType(obj interface{}) *k8s.EventTypes {
//	switch obj.(type) {
//	case *corev1.Service:
//		return getEventTypesByInformerKey(fsminformers.InformerKeyService)
//	case *mcsv1alpha1.ServiceImport:
//		return getEventTypesByInformerKey(fsminformers.InformerKeyServiceImport)
//	case *corev1.Endpoints:
//		return getEventTypesByInformerKey(fsminformers.InformerKeyEndpoints)
//	case *corev1.Secret:
//		return getEventTypesByInformerKey(fsminformers.InformerKeySecret)
//	case *networkingv1.Ingress:
//		return getEventTypesByInformerKey(fsminformers.InformerKeyK8sIngress)
//	case *networkingv1.IngressClass:
//		return getEventTypesByInformerKey(fsminformers.InformerKeyK8sIngressClass)
//	}
//
//	return nil
//}

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
	case fsminformers.InformerKeyK8sIngress:
		return &k8s.EventTypes{
			Add:    announcements.IngressAdded,
			Update: announcements.IngressUpdated,
			Delete: announcements.IngressDeleted,
		}
	case fsminformers.InformerKeyK8sIngressClass:
		return &k8s.EventTypes{
			Add:    announcements.IngressClassAdded,
			Update: announcements.IngressClassUpdated,
			Delete: announcements.IngressClassDeleted,
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

// Start starts the client
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
