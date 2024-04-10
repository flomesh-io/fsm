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
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
)

const (
	// K8SQuietPeriod is the time to wait for no service changes before syncing.
	K8SQuietPeriod = 1 * time.Second

	// K8SMaxPeriod is the maximum time to wait before forcing a sync, even
	// if there are active changes going on.
	K8SMaxPeriod = 5 * time.Second

	True  = "true"
	False = "false"
)

var (
	protocolHTTP = constants.ProtocolHTTP
	protocolGRPC = constants.ProtocolGRPC
)

// CtoKSyncer is the destination where services are registered.
//
// While in practice we only have one syncer (K8S), the interface abstraction
// makes it easy and possible to test the CtoKSource in isolation.
type CtoKSyncer struct {
	controller connector.ConnectController
	discClient connector.ServiceDiscoveryClient

	kubeClient kubernetes.Interface

	microAggregator Aggregator

	// ctx is used to cancel the CtoKSyncer.
	ctx context.Context

	fsmNamespace string

	// lock gates concurrent access to all the maps.
	lock sync.Mutex

	triggerCh chan struct{}
}

// NewCtoKSyncer creates a new mesh syncer
func NewCtoKSyncer(
	controller connector.ConnectController,
	discClient connector.ServiceDiscoveryClient,
	kubeClient kubernetes.Interface,
	ctx context.Context,
	fsmNamespace string) *CtoKSyncer {
	syncer := CtoKSyncer{
		controller:   controller,
		discClient:   discClient,
		kubeClient:   kubeClient,
		ctx:          ctx,
		fsmNamespace: fsmNamespace,
	}
	return &syncer
}

func (s *CtoKSyncer) SetMicroAggregator(microAggregator Aggregator) {
	s.microAggregator = microAggregator
}

// SetServices is called with the services that should be created.
// The key is the service name and the destination is the external DNS
// entry to point to.
func (s *CtoKSyncer) SetServices(svcs map[MicroSvcName]MicroSvcDomainName) {
	s.lock.Lock()
	defer s.lock.Unlock()

	lowercasedSvcs := make(map[string]string)
	for serviceName, serviceDNS := range svcs {
		lowercasedSvcs[strings.ToLower(string(serviceName))] = strings.ToLower(string(serviceDNS))
	}

	s.controller.GetC2KContext().SourceServices = lowercasedSvcs
	s.controller.GetC2KContext().RawServices = maps.Clone(lowercasedSvcs)
	s.trigger() // Any service change probably requires syncing
}

// Ready wait util ready
func (s *CtoKSyncer) Ready() {
	for {
		if ns, err := s.kubeClient.CoreV1().Namespaces().Get(s.ctx, s.namespace(), metav1.GetOptions{}); err == nil && ns != nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
}

// Informer implements the controller.Resource interface.
// It tells Kubernetes that we want to watch for changes to Services.
func (s *CtoKSyncer) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return s.kubeClient.CoreV1().Services(s.namespace()).List(s.ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return s.kubeClient.CoreV1().Services(s.namespace()).Watch(s.ctx, options)
			},
		},
		&apiv1.Service{},
		0,
		cache.Indexers{},
	)
}

// Upsert implements the controller.Resource interface.
func (s *CtoKSyncer) Upsert(key string, raw interface{}) error {
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
	s.controller.GetC2KContext().ServiceKeyToName[key] = service.Name
	s.controller.GetC2KContext().ServiceHashMap[service.Name] = s.serviceHash(service)

	// If the service is a Cloud-sourced service, then keep track of it
	// separately for a quick lookup.
	if service.Labels != nil && service.Labels[CloudSourcedServiceLabel] == True {
		s.controller.GetC2KContext().ServiceMapCache[service.Name] = service
		s.trigger() // Always trigger sync
	}

	log.Trace().Msgf("UpsertService, key:%s", key)
	return nil
}

// Delete implements the controller.Resource interface.
func (s *CtoKSyncer) Delete(key string, _ interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	name, ok := s.controller.GetC2KContext().ServiceKeyToName[key]
	if !ok {
		// This is a weird scenario, but in unit tests we've seen this happen
		// in cases where the delete happens very quickly after the create.
		// Just to be sure, lets trigger a sync. This is cheap cause it'll
		// do nothing if there is no work to be done.
		s.trigger()
		return nil
	}

	delete(s.controller.GetC2KContext().ServiceKeyToName, key)
	delete(s.controller.GetC2KContext().ServiceMapCache, name)
	delete(s.controller.GetC2KContext().ServiceHashMap, name)

	// If the service that is deleted is part of cloud services, then
	// we need to trigger a sync to recreate it.
	if _, ok = s.controller.GetC2KContext().SourceServices[name]; ok {
		s.trigger()
	}

	log.Info().Msgf("delete service, key:%s name:%s", key, name)
	return nil
}

// Run implements the controller.Backgrounder interface.
func (s *CtoKSyncer) Run(ch <-chan struct{}) {
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
			coalesce(s.ctx,
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

		svcClient := s.kubeClient.CoreV1().Services(s.namespace())
		for _, name := range deletes {
			if err := svcClient.Delete(s.ctx, name, metav1.DeleteOptions{}); err != nil {
				log.Warn().Msgf("warn deleting service, name:%s warn:%v", name, err)
			}
		}

		for _, svc := range creates {
			if _, err := svcClient.Create(s.ctx, svc, metav1.CreateOptions{}); err != nil {
				log.Error().Msgf("creating service, name:%s error:%v", svc.Name, err)
			}
		}
	}
}

// crudList returns the services to create, update, and delete (respectively).
func (s *CtoKSyncer) crudList() ([]*apiv1.Service, []string) {
	var createSvcs []*apiv1.Service
	var deleteSvcs []string
	extendServices := make(map[string]string, 0)
	ipFamilyPolicy := apiv1.IPFamilyPolicySingleStack
	// Determine what needs to be created or updated
	for cloudName, cloudDNS := range s.controller.GetC2KContext().SourceServices {
		svcMetaMap := s.microAggregator.Aggregate(s.ctx, MicroSvcName(cloudName))
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
			preHv := s.controller.GetC2KContext().ServiceHashMap[string(microSvcName)]
			// If this is an already registered service, then update it
			if svc, ok := s.controller.GetC2KContext().ServiceMapCache[string(microSvcName)]; ok {
				if svc.Spec.ExternalName == cloudDNS {
					// Matching service, no update required.
					continue
				}

				svc.ObjectMeta.Annotations = map[string]string{
					// Ensure we don't sync the service back to cloud
					connector.AnnotationMeshServiceSync:           string(s.discClient.MicroServiceProvider()),
					connector.AnnotationCloudServiceInheritedFrom: cloudName,
				}
				if s.controller.GetC2KWithGateway() {
					if s.discClient.IsInternalServices() {
						svc.ObjectMeta.Annotations[connector.AnnotationMeshServiceInternalSync] = True
					}
				}
				if svcMeta.HealthCheck {
					svc.ObjectMeta.Annotations[connector.AnnotationServiceSyncK8sToFgw] = False
					svc.ObjectMeta.Annotations[connector.AnnotationServiceSyncK8sToCloud] = False
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
						connector.AnnotationMeshServiceSync:           string(s.discClient.MicroServiceProvider()),
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
			if s.controller.GetC2KWithGateway() {
				if s.discClient.IsInternalServices() {
					createSvc.ObjectMeta.Annotations[connector.AnnotationMeshServiceInternalSync] = True
				}
			}
			if svcMeta.HealthCheck {
				createSvc.ObjectMeta.Annotations[connector.AnnotationServiceSyncK8sToFgw] = False
				createSvc.ObjectMeta.Annotations[connector.AnnotationServiceSyncK8sToCloud] = False
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
			s.controller.GetC2KContext().SourceServices[cloudName] = cloudDNS
		}
	}

	// Determine what needs to be deleted
	for k := range s.controller.GetC2KContext().ServiceMapCache {
		if _, ok := s.controller.GetC2KContext().SourceServices[k]; !ok {
			deleteSvcs = append(deleteSvcs, k)
		}
	}

	return createSvcs, deleteSvcs
}

func (s *CtoKSyncer) fillService(svcMeta *MicroSvcMeta, createSvc *apiv1.Service) {
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
	if len(svcMeta.ViaGateway) > 0 {
		createSvc.ObjectMeta.Annotations[connector.AnnotationCloudServiceViaGateway] = svcMeta.ViaGateway
	}
	if len(svcMeta.ViaGateway) > 0 && !strings.EqualFold(svcMeta.ClusterSet, s.controller.GetClusterSet()) {
		createSvc.ObjectMeta.Annotations[connector.AnnotationCloudServiceClusterSet] = svcMeta.ClusterSet
		delete(createSvc.ObjectMeta.Annotations, connector.AnnotationMeshServiceInternalSync)
	}
	for addr := range svcMeta.Addresses {
		createSvc.ObjectMeta.Annotations[fmt.Sprintf("%s-%d", connector.AnnotationMeshEndpointAddr, utils.IP2Int(addr.To4()))] = fmt.Sprintf("%v", ports)
	}
}

func (s *CtoKSyncer) existPort(svc *apiv1.Service, port MicroSvcPort, appProtocol MicroSvcAppProtocol) bool {
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
func (s *CtoKSyncer) namespace() string {
	if deriveNamespace := s.controller.GetDeriveNamespace(); len(deriveNamespace) > 0 {
		return deriveNamespace
	}

	// Default to the default namespace. This should not be "all" since we
	// want a specific namespace to watch and write to.
	return metav1.NamespaceDefault
}

// trigger will notify a sync should occur. lock must be held.
//
// This is not synchronous and does not guarantee a sync will happen. This
// just sends a notification that a sync is likely necessary.
func (s *CtoKSyncer) trigger() {
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

func (s *CtoKSyncer) serviceHash(service *apiv1.Service) uint64 {
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
