package ktog

import (
	"context"
	"strconv"
	"sync"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/connector"
)

// ServiceResource implements controller.Resource to sync CatalogService resource
// types from K8S.
type ServiceResource struct {
	FsmNamespace  string
	Client        kubernetes.Interface
	GatewayClient gwapi.Interface

	servicesInformer cache.SharedIndexInformer

	// Ctx is used to cancel processes kicked off by ServiceResource.
	Ctx context.Context

	// AllowK8sNamespacesSet is a set of k8s namespaces to explicitly allow for
	// syncing. It supports the special character `*` which indicates that
	// all k8s namespaces are eligible unless explicitly denied. This filter
	// is applied before checking pod annotations.
	AllowK8sNamespacesSet mapset.Set

	// DenyK8sNamespacesSet is a set of k8s namespaces to explicitly deny
	// syncing and thus Service registration with Consul. An empty set
	// means that no namespaces are removed from consideration. This filter
	// takes precedence over AllowK8sNamespacesSet.
	DenyK8sNamespacesSet mapset.Set

	// ExplictEnable should be set to true to require explicit enabling
	// using annotations. If this is false, then services are implicitly
	// enabled (aka default enabled).
	ExplicitEnable bool

	// serviceLock must be held for any read/write to these maps.
	serviceLock sync.RWMutex

	// serviceMap holds services we should sync to gateway. Keys are the
	// in the form <kube namespace>/<kube svc name>.
	serviceMap map[string]*corev1.Service

	GatewayResource *GatewayResource

	Syncer Syncer
}

// Informer implements the controller.Resource interface.
func (t *ServiceResource) Informer() cache.SharedIndexInformer {
	// Watch all k8s namespaces. Events will be filtered out as appropriate
	// based on the allow and deny lists in the `shouldSync` function.
	if t.servicesInformer == nil {
		t.servicesInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return t.Client.CoreV1().Services(metav1.NamespaceAll).List(t.Ctx, options)
				},

				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return t.Client.CoreV1().Services(metav1.NamespaceAll).Watch(t.Ctx, options)
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
func (t *ServiceResource) Run(ch <-chan struct{}) {
	t.serviceLock.Lock()
	defer t.serviceLock.Unlock()
	if svcList, err := t.Client.CoreV1().Services(metav1.NamespaceAll).List(t.Ctx, metav1.ListOptions{}); err == nil {
		for _, svc := range svcList.Items {
			shaddowSvc := svc
			if key, err := cache.MetaNamespaceKeyFunc(svc); err == nil {
				if t.serviceMap == nil {
					t.serviceMap = make(map[string]*corev1.Service)
				}
				if !t.shouldSync(&shaddowSvc) {
					continue
				}
				t.serviceMap[key] = &shaddowSvc
			}
		}
	}
}

// Upsert implements the controller.Resource interface.
func (t *ServiceResource) Upsert(key string, raw interface{}) error {
	// We expect a CatalogService. If it isn't a Service then just ignore it.
	service, ok := raw.(*corev1.Service)
	if !ok {
		log.Warn().Msgf("upsert got invalid type raw:%v", raw)
		return nil
	}

	t.serviceLock.Lock()
	defer t.serviceLock.Unlock()

	if t.serviceMap == nil {
		t.serviceMap = make(map[string]*corev1.Service)
	}

	if !t.shouldSync(service) {
		// Check if its in our map and delete it.
		if _, ok = t.serviceMap[key]; ok {
			log.Info().Msgf("Service should no longer be synced Service:%s", key)
			t.doDelete(key)
		} else {
			log.Debug().Msgf("[ServiceResource.Upsert] syncing disabled for Service, ignoring key:%s", key)
		}
		return nil
	}

	// Syncing is enabled, let's keep track of this Service.
	t.serviceMap[key] = service
	log.Debug().Msgf("[ServiceResource.Upsert] adding Service to serviceMap key:%s Service:%v", key, service)

	t.sync()
	log.Info().Msgf("upsert key:%s", key)
	return nil
}

// Delete implements the controller.Resource interface.
func (t *ServiceResource) Delete(key string, _ interface{}) error {
	t.serviceLock.Lock()
	defer t.serviceLock.Unlock()
	t.doDelete(key)
	log.Info().Msgf("delete key:%s", key)
	return nil
}

// doDelete is a helper function for deletion.
//
// Precondition: assumes t.serviceLock is held.
func (t *ServiceResource) doDelete(key string) {
	delete(t.serviceMap, key)
	log.Debug().Msgf("[doDelete] deleting Service from serviceMap key:%s", key)
	t.sync()
}

// shouldSync returns true if resyncing should be enabled for the given Service.
func (t *ServiceResource) shouldSync(svc *corev1.Service) bool {
	// Namespace logic
	// If in deny list, don't sync
	if t.DenyK8sNamespacesSet.Contains(svc.Namespace) {
		log.Debug().Msgf("[shouldSync] Service is in the deny list svc.Namespace:%s Service:%v", svc.Namespace, svc)
		return false
	}

	// If not in allow list or allow list is not *, don't sync
	if !t.AllowK8sNamespacesSet.Contains("*") && !t.AllowK8sNamespacesSet.Contains(svc.Namespace) {
		log.Debug().Msgf("[shouldSync] Service not in allow list svc.Namespace:%s Service:%v", svc.Namespace, svc)
		return false
	}

	raw, ok := svc.Annotations[connector.AnnotationServiceSyncK8sToFgw]
	if !ok {
		// If there is no explicit value, then set it to our current default.
		return !t.ExplicitEnable
	}

	v, err := strconv.ParseBool(raw)
	if err != nil {
		log.Warn().Msgf("error parsing Service-sync annotation Service-name:%s %s err:%v",
			svc.Name, svc.Namespace,
			err)

		// Fallback to default
		return !t.ExplicitEnable
	}

	return v
}

// sync calls the Syncer.Sync function from the generated registrations.
//
// Precondition: lock must be held.
func (t *ServiceResource) sync() {
	// NOTE(mitchellh): This isn't the most efficient way to do this and
	// the times that sync are called are also not the most efficient. All
	// of these are implementation details so lets improve this later when
	// it becomes a performance issue and just do the easy thing first.
	rs := make([]*corev1.Service, 0, len(t.serviceMap))
	for _, svc := range t.serviceMap {
		rs = append(rs, svc)
	}

	// Sync, which should be non-blocking in real-world cases
	t.Syncer.Sync(rs)
}
