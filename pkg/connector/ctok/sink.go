package ctok

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/maps"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
)

const (
	// K8SQuietPeriod is the time to wait for no service changes before syncing.
	K8SQuietPeriod = 1 * time.Second

	// K8SMaxPeriod is the maximum time to wait before forcing a sync, even
	// if there are active changes going on.
	K8SMaxPeriod = 5 * time.Second

	True = "true"
)

var (
	protocolHTTP = constants.ProtocolHTTP
	protocolGRPC = constants.ProtocolGRPC
)

// NewSink creates a new mesh sink
func NewSink(ctx context.Context, kubeClient kubernetes.Interface, discClient provider.ServiceDiscoveryClient, fsmNamespace string) *Sink {
	sink := Sink{
		Ctx:              ctx,
		KubeClient:       kubeClient,
		DiscClient:       discClient,
		fsmNamespace:     fsmNamespace,
		serviceKeyToName: make(map[string]string),
		serviceMapCache:  make(map[string]*apiv1.Service),
		serviceHashMap:   make(map[string]uint64),
	}
	return &sink
}

// Sink is the destination where services are registered.
//
// While in practice we only have one sink (K8S), the interface abstraction
// makes it easy and possible to test the Source in isolation.
type Sink struct {
	fsmNamespace string

	KubeClient kubernetes.Interface

	DiscClient provider.ServiceDiscoveryClient

	MicroAggregator Aggregator

	// SyncPeriod is the duration to wait between registering or deregistering
	// services in Kubernetes. This can be fairly short since no work will be
	// done if there are no changes.
	SyncPeriod time.Duration

	// Ctx is used to cancel the Sink.
	Ctx context.Context

	// lock gates concurrent access to all the maps.
	lock sync.Mutex

	// sourceServices holds cloud services that should be synced to Kube.
	// It maps from cloud service names to cloud DNS entry, e.g.
	// We lowercase the cloud service names and DNS entries
	// because Kube names must be lowercase.
	sourceServices map[string]string
	rawServices    map[string]string

	// serviceKeyToName maps from Kube controller keys to Kube service names.
	// Controller keys are in the form <kube namespace>/<kube svc name>
	// e.g. default/foo, and are the keys Kube uses to inform that something
	// changed.
	serviceKeyToName map[string]string

	// serviceMapCache is a subset of serviceMap. It holds all Kube services
	// that were created by this sync process. Keys are Kube service names.
	// It's populated from Kubernetes data.
	serviceMapCache map[string]*apiv1.Service

	serviceHashMap map[string]uint64

	triggerCh chan struct{}
}

// SetServices is called with the services that should be created.
// The key is the service name and the destination is the external DNS
// entry to point to.
func (s *Sink) SetServices(svcs map[MicroSvcName]MicroSvcDomainName) {
	s.lock.Lock()
	defer s.lock.Unlock()

	lowercasedSvcs := make(map[string]string)
	for serviceName, serviceDNS := range svcs {
		lowercasedSvcs[strings.ToLower(string(serviceName))] = strings.ToLower(string(serviceDNS))
	}

	s.sourceServices = lowercasedSvcs
	s.rawServices = maps.Clone(lowercasedSvcs)
	s.trigger() // Any service change probably requires syncing
}

// Ready wait util ready
func (s *Sink) Ready() {
	for {
		if ns, err := s.KubeClient.CoreV1().Namespaces().Get(s.Ctx, s.namespace(), metav1.GetOptions{}); err == nil && ns != nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
}

// Informer implements the controller.Resource interface.
// It tells Kubernetes that we want to watch for changes to Services.
func (s *Sink) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return s.KubeClient.CoreV1().Services(s.namespace()).List(s.Ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return s.KubeClient.CoreV1().Services(s.namespace()).Watch(s.Ctx, options)
			},
		},
		&apiv1.Service{},
		0,
		cache.Indexers{},
	)
}

// Upsert implements the controller.Resource interface.
func (s *Sink) Upsert(key string, raw interface{}) error {
	// We expect a Service. If it isn't a service then just ignore it.
	service, ok := raw.(*apiv1.Service)
	if !ok {
		log.Warn().Msgf("UpsertService got invalid type, raw:%v", raw)
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// Store all the key to name mappings. We need this because the key
	// is opaque but we want to do all the lookups by service name.
	s.serviceKeyToName[key] = service.Name
	s.serviceHashMap[service.Name] = s.serviceHash(service)

	// If the service is a Cloud-sourced service, then keep track of it
	// separately for a quick lookup.
	if service.Labels != nil && service.Labels[CloudSourcedServiceLabel] == True {
		s.serviceMapCache[service.Name] = service
		s.trigger() // Always trigger sync
	}

	log.Trace().Msgf("UpsertService, key:%s", key)
	return nil
}

// Delete implements the controller.Resource interface.
func (s *Sink) Delete(key string, _ interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	name, ok := s.serviceKeyToName[key]
	if !ok {
		// This is a weird scenario, but in unit tests we've seen this happen
		// in cases where the delete happens very quickly after the create.
		// Just to be sure, lets trigger a sync. This is cheap cause it'll
		// do nothing if there is no work to be done.
		s.trigger()
		return nil
	}

	delete(s.serviceKeyToName, key)
	delete(s.serviceMapCache, name)
	delete(s.serviceHashMap, name)

	// If the service that is deleted is part of cloud services, then
	// we need to trigger a sync to recreate it.
	if _, ok = s.sourceServices[name]; ok {
		s.trigger()
	}

	log.Info().Msgf("delete service, key:%s name:%s", key, name)
	return nil
}

// Run implements the controller.Backgrounder interface.
func (s *Sink) Run(ch <-chan struct{}) {
	log.Info().Msg("starting runner for syncing")

	// Initialize the trigger channel. We send an initial message so that
	// our loop below runs immediately.
	s.lock.Lock()
	var triggerCh chan struct{}
	if s.triggerCh == nil {
		triggerCh = make(chan struct{}, 1)
		triggerCh <- struct{}{}
		s.triggerCh = triggerCh
	}
	s.lock.Unlock()

	for {
		select {
		case <-ch:
			return
		case <-triggerCh:
			// Coalesce to prevent lots of API calls during churn periods.
			coalesce(s.Ctx,
				K8SQuietPeriod, K8SMaxPeriod,
				func(ctx context.Context) {
					select {
					case <-triggerCh:
					case <-ctx.Done():
					}
				})
		}

		s.lock.Lock()
		creates, deletes := s.crudList()
		s.lock.Unlock()
		if len(creates) > 0 || len(deletes) > 0 {
			log.Info().Msgf("sync triggered, create:%d delete:%d", len(creates), len(deletes))
		}

		svcClient := s.KubeClient.CoreV1().Services(s.namespace())
		for _, name := range deletes {
			if err := svcClient.Delete(s.Ctx, name, metav1.DeleteOptions{}); err != nil {
				log.Warn().Msgf("warn deleting service, name:%s warn:%v", name, err)
			}
		}

		for _, svc := range creates {
			if _, err := svcClient.Create(s.Ctx, svc, metav1.CreateOptions{}); err != nil {
				log.Error().Msgf("creating service, name:%s error:%v", svc.Name, err)
			}
		}
	}
}

// crudList returns the services to create, update, and delete (respectively).
func (s *Sink) crudList() ([]*apiv1.Service, []string) {
	var createSvcs []*apiv1.Service
	var deleteSvcs []string
	extendServices := make(map[string]string, 0)
	ipFamilyPolicy := apiv1.IPFamilyPolicySingleStack
	// Determine what needs to be created or updated
	for cloudName, cloudDNS := range s.sourceServices {
		svcMetaMap := s.MicroAggregator.Aggregate(MicroSvcName(cloudName), MicroSvcDomainName(cloudDNS))
		if len(svcMetaMap) == 0 {
			continue
		}
		for microSvcName, svcMeta := range svcMetaMap {
			if len(svcMeta.Addresses) == 0 {
				continue
			}
			if !strings.EqualFold(string(microSvcName), cloudName) {
				extendServices[string(microSvcName)] = cloudDNS
			}
			preHv := s.serviceHashMap[string(microSvcName)]
			// If this is an already registered service, then update it
			if svc, ok := s.serviceMapCache[string(microSvcName)]; ok {
				if svc.Spec.ExternalName == cloudDNS {
					// Matching service, no update required.
					continue
				}

				svc.ObjectMeta.Annotations = map[string]string{
					// Ensure we don't sync the service back to cloud
					connector.AnnotationMeshServiceSync:           s.DiscClient.MicroServiceProvider(),
					connector.AnnotationCloudServiceInheritedFrom: cloudName,
				}
				if withGateway {
					if s.DiscClient.IsInternalServices() {
						svc.ObjectMeta.Annotations[connector.AnnotationMeshServiceInternalSync] = True
					}
				}
				s.fillService(svcMeta, svc)
				if preHv == s.serviceHash(svc) {
					log.Trace().Msgf("service already registered in K8S, not registering, name:%s", string(microSvcName))
					continue
				}
				deleteSvcs = append(deleteSvcs, string(microSvcName))
				continue
			}

			// Register!
			createSvc := &apiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:   string(microSvcName),
					Labels: map[string]string{CloudSourcedServiceLabel: True},
					Annotations: map[string]string{
						// Ensure we don't sync the service back to Cloud
						connector.AnnotationMeshServiceSync:           s.DiscClient.MicroServiceProvider(),
						connector.AnnotationCloudServiceInheritedFrom: cloudName,
					},
				},

				Spec: apiv1.ServiceSpec{
					Type:      apiv1.ServiceTypeClusterIP,
					ClusterIP: apiv1.ClusterIPNone,
					Selector: map[string]string{
						CloudServiceLabel: string(microSvcName),
					},
					IPFamilies:     []apiv1.IPFamily{apiv1.IPv4Protocol},
					IPFamilyPolicy: &ipFamilyPolicy,
				},
			}
			if withGateway {
				if s.DiscClient.IsInternalServices() {
					createSvc.ObjectMeta.Annotations[connector.AnnotationMeshServiceInternalSync] = True
				}
			}
			s.fillService(svcMeta, createSvc)
			if preHv == s.serviceHash(createSvc) {
				log.Debug().Msgf("service already registered in K8S, not registering, name:%s", string(microSvcName))
				continue
			}

			createSvcs = append(createSvcs, createSvc)
		}
	}

	if len(extendServices) > 0 {
		for cloudName, cloudDNS := range extendServices {
			s.sourceServices[cloudName] = cloudDNS
		}
	}

	// Determine what needs to be deleted
	for k := range s.serviceMapCache {
		if _, ok := s.sourceServices[k]; !ok {
			deleteSvcs = append(deleteSvcs, k)
		}
	}

	return createSvcs, deleteSvcs
}

func (s *Sink) fillService(svcMeta *MicroSvcMeta, createSvc *apiv1.Service) {
	ports := make([]int, 0)
	for port, appProtocol := range svcMeta.Ports {
		if exists := s.existPort(createSvc, MicroSvcPort(port), appProtocol); !exists {
			specPort := apiv1.ServicePort{
				Name:       fmt.Sprintf("%s%d", appProtocol, port),
				Protocol:   apiv1.ProtocolTCP,
				Port:       int32(port),
				TargetPort: intstr.FromInt(int(port)),
			}
			if appProtocol == constants.ProtocolHTTP {
				specPort.AppProtocol = &protocolHTTP
			}
			if appProtocol == constants.ProtocolGRPC {
				specPort.AppProtocol = &protocolGRPC
			}
			createSvc.Spec.Ports = append(createSvc.Spec.Ports, specPort)
		}
		ports = append(ports, int(port))
	}
	sort.Ints(ports)
	createSvc.ObjectMeta.Annotations[connector.AnnotationCloudServiceInheritedClusterID] = svcMeta.ClusterId
	for addr := range svcMeta.Addresses {
		createSvc.ObjectMeta.Annotations[fmt.Sprintf("%s-%d", connector.AnnotationMeshEndpointAddr, utils.IP2Int(addr.To4()))] = fmt.Sprintf("%v", ports)
	}
}

func (s *Sink) existPort(svc *apiv1.Service, port MicroSvcPort, appProtocol MicroSvcAppProtocol) bool {
	if len(svc.Spec.Ports) > 0 {
		for _, specPort := range svc.Spec.Ports {
			if specPort.Port == int32(port) ||
				specPort.Name == fmt.Sprintf("%s%d", appProtocol, port) {
				return true
			}
		}
	}
	return false
}

// namespace returns the K8S namespace to setup the resource watchers in.
func (s *Sink) namespace() string {
	if syncCloudNamespace != "" {
		return syncCloudNamespace
	}

	// Default to the default namespace. This should not be "all" since we
	// want a specific namespace to watch and write to.
	return metav1.NamespaceDefault
}

// trigger will notify a sync should occur. lock must be held.
//
// This is not synchronous and does not guarantee a sync will happen. This
// just sends a notification that a sync is likely necessary.
func (s *Sink) trigger() {
	if s.triggerCh != nil {
		// Non-blocking send. This is okay because we always buffer triggerCh
		// to one. So if this blocks it means that a message is already waiting
		// which is equivalent to us sending the trigger. No information loss!
		select {
		case s.triggerCh <- struct{}{}:
		default:
		}
	}
}

func (s *Sink) serviceHash(service *apiv1.Service) uint64 {
	bytes := make([]byte, 0)
	if len(service.Labels) > 0 {
		labelBytes, _ := json.Marshal(service.Labels)
		bytes = append(bytes, labelBytes...)
	}
	if len(service.Annotations) > 0 {
		annoBytes, _ := json.Marshal(service.Annotations)
		bytes = append(bytes, annoBytes...)
	}
	if len(service.Spec.Ports) > 0 {
		portBytes, _ := json.Marshal(service.Spec.Ports)
		bytes = append(bytes, portBytes...)
	}
	return utils.Hash(bytes)
}
