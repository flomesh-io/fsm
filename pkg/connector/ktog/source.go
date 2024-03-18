package ktog

import (
	"context"
	"strconv"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("connector-k2g")
)

// KtoGSource implements controller.Resource to sync Service source
// types from K8S.
type KtoGSource struct {
	controller    connector.ConnectController
	syncer        Syncer
	gatewaySource *GatewaySource

	fsmNamespace  string
	kubeClient    kubernetes.Interface
	gatewayClient gwapi.Interface

	// ctx is used to cancel processes kicked off by KtoGSource.
	ctx context.Context

	// serviceLock must be held for any read/write to these maps.
	serviceLock sync.RWMutex

	servicesInformer cache.SharedIndexInformer
}

func NewKtoGSource(controller connector.ConnectController, syncer Syncer,
	gatewaySource *GatewaySource,
	fsmNamespace string,
	kubeClient kubernetes.Interface,
	gatewayClient gwapi.Interface,
	ctx context.Context) *KtoGSource {
	return &KtoGSource{
		controller:    controller,
		syncer:        syncer,
		gatewaySource: gatewaySource,
		fsmNamespace:  fsmNamespace,
		kubeClient:    kubeClient,
		gatewayClient: gatewayClient,
		ctx:           ctx,
	}
}

// Informer implements the controller.Resource interface.
func (t *KtoGSource) Informer() cache.SharedIndexInformer {
	// Watch all k8s namespaces. Events will be filtered out as appropriate
	// based on the allow and deny lists in the `shouldSync` function.
	if t.servicesInformer == nil {
		t.servicesInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return t.kubeClient.CoreV1().Services(metav1.NamespaceAll).List(t.ctx, options)
				},

				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return t.kubeClient.CoreV1().Services(metav1.NamespaceAll).Watch(t.ctx, options)
				},
			},
			&corev1.Service{},
			0,
			cache.Indexers{},
		)
	}
	return t.servicesInformer
}

// Run implements the controller.Backgrounder interface.
func (t *KtoGSource) Run(ch <-chan struct{}) {
	t.serviceLock.Lock()
	defer t.serviceLock.Unlock()
	if svcList, err := t.kubeClient.CoreV1().Services(metav1.NamespaceAll).List(t.ctx, metav1.ListOptions{}); err == nil {
		for _, svc := range svcList.Items {
			shaddowSvc := svc
			if key, err := cache.MetaNamespaceKeyFunc(svc); err == nil {
				if !t.shouldSync(&shaddowSvc) {
					continue
				}
				t.controller.GetK2GContext().ServiceMap[key] = &shaddowSvc
			}
		}
	}
}

// Upsert implements the controller.Resource interface.
func (t *KtoGSource) Upsert(key string, raw interface{}) error {
	// We expect a Service. If it isn't a Service then just ignore it.
	service, ok := raw.(*corev1.Service)
	if !ok {
		log.Warn().Msgf("upsert got invalid type raw:%v", raw)
		return nil
	}

	t.serviceLock.Lock()
	defer t.serviceLock.Unlock()

	if !t.shouldSync(service) {
		// Check if its in our map and delete it.
		if _, ok = t.controller.GetK2GContext().ServiceMap[key]; ok {
			log.Info().Msgf("Service should no longer be synced Service:%s", key)
			t.doDelete(key)
		} else {
			log.Debug().Msgf("[KtoGSource.Upsert] syncing disabled for Service, ignoring key:%s", key)
		}
		return nil
	}

	// Syncing is enabled, let's keep track of this Service.
	t.controller.GetK2GContext().ServiceMap[key] = service
	log.Debug().Msgf("[KtoGSource.Upsert] adding Service to serviceMap key:%s Service:%v", key, service)

	t.sync()
	log.Info().Msgf("upsert key:%s", key)
	return nil
}

// Delete implements the controller.Resource interface.
func (t *KtoGSource) Delete(key string, _ interface{}) error {
	t.serviceLock.Lock()
	defer t.serviceLock.Unlock()
	t.doDelete(key)
	log.Info().Msgf("delete key:%s", key)
	return nil
}

// doDelete is a helper function for deletion.
//
// Precondition: assumes t.serviceLock is held.
func (t *KtoGSource) doDelete(key string) {
	delete(t.controller.GetK2GContext().ServiceMap, key)
	log.Debug().Msgf("[doDelete] deleting Service from serviceMap key:%s", key)
	t.sync()
}

// shouldSync returns true if resyncing should be enabled for the given Service.
func (t *KtoGSource) shouldSync(svc *corev1.Service) bool {
	// Namespace logic
	// If in deny list, don't sync
	if t.controller.GetK2GDenyK8SNamespaceSet().Contains(svc.Namespace) {
		log.Debug().Msgf("[shouldSync] Service is in the deny list svc.Namespace:%s Service:%v", svc.Namespace, svc)
		return false
	}

	// If not in allow list or allow list is not *, don't sync
	if !t.controller.GetK2GAllowK8SNamespaceSet().Contains("*") && !t.controller.GetK2GAllowK8SNamespaceSet().Contains(svc.Namespace) {
		log.Debug().Msgf("[shouldSync] Service not in allow list svc.Namespace:%s Service:%v", svc.Namespace, svc)
		return false
	}

	raw, ok := svc.Annotations[connector.AnnotationServiceSyncK8sToFgw]
	if !ok {
		// If there is no explicit value, then set it to our current default.
		return t.controller.GetK2GDefaultSync()
	}

	v, err := strconv.ParseBool(raw)
	if err != nil {
		log.Warn().Msgf("error parsing Service-sync annotation Service-name:%s %s err:%v",
			svc.Name, svc.Namespace,
			err)

		// Fallback to default
		return t.controller.GetK2GDefaultSync()
	}

	return v
}

// sync calls the Syncer.Sync function from the generated registrations.
//
// Precondition: lock must be held.
func (t *KtoGSource) sync() {
	// NOTE(mitchellh): This isn't the most efficient way to do this and
	// the times that sync are called are also not the most efficient. All
	// of these are implementation details so lets improve this later when
	// it becomes a performance issue and just do the easy thing first.
	rs := make([]*corev1.Service, 0, len(t.controller.GetK2GContext().ServiceMap))
	for _, svc := range t.controller.GetK2GContext().ServiceMap {
		rs = append(rs, svc)
	}

	// Sync, which should be non-blocking in real-world cases
	t.syncer.Sync(rs)
}
