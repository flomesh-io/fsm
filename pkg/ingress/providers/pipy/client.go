package pipy

import (
	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes"
	k8scache "k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/cache"
	"github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/repo"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

var (
	log = logger.New("controller-gatewayapi")
)

// NewIngressController returns a ingress.Controller interface related to functionality provided by the resources in the k8s ingress API group
func NewIngressController(informerCollection *fsminformers.InformerCollection, kubeClient kubernetes.Interface, msgBroker *messaging.Broker, cfg configurator.Configurator, certMgr *certificate.Manager) Controller {
	return newClient(informerCollection, kubeClient, msgBroker, cfg, certMgr)
}

func newClient(informerCollection *fsminformers.InformerCollection, kubeClient kubernetes.Interface, msgBroker *messaging.Broker, cfg configurator.Configurator, _ *certificate.Manager) *client {
	c := &client{
		informers:  informerCollection,
		kubeClient: kubeClient,
		msgBroker:  msgBroker,
		cfg:        cfg,
		cache:      cache.NewCache(kubeClient, informerCollection, cfg),
	}

	// Initialize informers
	for _, informerKey := range []fsminformers.InformerKey{
		fsminformers.InformerKeyService,
		fsminformers.InformerKeyServiceImport,
		fsminformers.InformerKeyEndpoints,
		fsminformers.InformerKeySecret,
		fsminformers.InformerKeyK8sIngressClass,
		fsminformers.InformerKeyK8sIngress,
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
