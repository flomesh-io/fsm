package ctok

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

const (
	// K8SQuietPeriod is the time to wait for no service changes before syncing.
	K8SQuietPeriod = messaging.ConnectorUpdateMaxWindow

	// K8SMaxPeriod is the maximum time to wait before forcing a sync, even
	// if there are active changes going on.
	K8SMaxPeriod = messaging.ConnectorUpdateMaxWindow

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

	informer cache.SharedIndexInformer

	microAggregator Aggregator

	fillEndpoints bool

	// ctx is used to cancel the CtoKSyncer.
	ctx context.Context

	fsmNamespace string

	// lock gates concurrent access to all the maps.
	lock sync.Mutex

	triggerCh chan struct{}

	syncWorkQueues *workerpool.WorkerPool
}

// NewCtoKSyncer creates a new mesh syncer
func NewCtoKSyncer(
	controller connector.ConnectController,
	discClient connector.ServiceDiscoveryClient,
	kubeClient kubernetes.Interface,
	ctx context.Context,
	fsmNamespace string,
	nWorkers uint) *CtoKSyncer {
	syncer := CtoKSyncer{
		controller:     controller,
		discClient:     discClient,
		kubeClient:     kubeClient,
		ctx:            ctx,
		fsmNamespace:   fsmNamespace,
		syncWorkQueues: workerpool.NewWorkerPool(int(nWorkers)),
	}
	return &syncer
}

func (s *CtoKSyncer) SetMicroAggregator(microAggregator Aggregator) {
	s.microAggregator = microAggregator
}

// SetServices is called with the services that should be created.
// The key is the service name and the destination is the external DNS
// entry to point to.
func (s *CtoKSyncer) SetServices(svcs map[connector.KubeSvcName]connector.ServiceConversion, catalogServices []ctv1.NamespacedService) {
	s.lock.Lock()
	defer s.lock.Unlock()

	sourceServices := make(map[connector.KubeSvcName]connector.CloudSvcName, len(svcs))
	nativeServices := make(map[connector.KubeSvcName]connector.CloudSvcName, len(svcs))
	externalServices := make(map[connector.CloudSvcName]connector.ExternalName, len(svcs))
	for kubeSvcName, serviceConversion := range svcs {
		sourceServices[kubeSvcName] = serviceConversion.Service
		nativeServices[kubeSvcName] = serviceConversion.Service
		if len(serviceConversion.ExternalName) > 0 {
			externalServices[serviceConversion.Service] = serviceConversion.ExternalName
		}
	}

	s.controller.GetC2KContext().SourceServices = sourceServices
	s.controller.GetC2KContext().NativeServices = nativeServices
	s.controller.GetC2KContext().ExternalServices = externalServices

	hash := uint64(0)
	if len(catalogServices) > 0 {
		sort.Sort(ctv1.NamespacedServiceSlice(catalogServices))
		hash, _ = hashstructure.Hash(catalogServices, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			})
	}
	if s.controller.GetC2KContext().CatalogServicesHash != hash {
		s.controller.GetC2KContext().CatalogServices = catalogServices
		s.controller.GetC2KContext().CatalogServicesHash = hash
	}

	s.trigger() // Any service change probably requires syncing
}

// Ready wait util ready
func (s *CtoKSyncer) Ready() {
	for {
		if ns, err := s.kubeClient.CoreV1().Namespaces().Get(s.ctx, s.namespace(), metav1.GetOptions{}); err == nil && ns != nil {
			break
		} else if err != nil {
			log.Error().Msgf("get namespace:%s error:%v", s.namespace(), err)
		} else if ns == nil {
			log.Error().Msgf("can't get namespace:%s", s.namespace())
		}
		time.Sleep(5 * time.Second)
	}
}

// Informer implements the controller.Resource interface.
// It tells Kubernetes that we want to watch for changes to Services.
func (s *CtoKSyncer) Informer() cache.SharedIndexInformer {
	if s.informer == nil {
		s.informer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					serviceList, err := s.kubeClient.CoreV1().Services(s.namespace()).List(s.ctx, options)
					if err != nil {
						log.Error().Msgf("cache.NewSharedIndexInformer Services ListFunc:%v", err)
					}
					return serviceList, err
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					service, err := s.kubeClient.CoreV1().Services(s.namespace()).Watch(s.ctx, options)
					if err != nil {
						log.Error().Msgf("cache.NewSharedIndexInformer Services WatchFunc:%v", err)
					}
					return service, err
				},
			},
			&corev1.Service{},
			0,
			cache.Indexers{},
		)
	}
	return s.informer
}

// Upsert implements the controller.Resource interface.
func (s *CtoKSyncer) Upsert(key string, raw interface{}) error {
	// We expect a Service. If it isn't a service then just ignore it.
	service, ok := raw.(*corev1.Service)
	if !ok {
		log.Warn().Msgf("UpsertService got invalid type, raw:%v", raw)
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// Store all the key to name mappings. We need this because the key
	// is opaque but we want to do all the lookups by service name.
	s.controller.GetC2KContext().KubeServiceCache[connector.KubeSvcKey(key)] = service

	// If the service is a Cloud-sourced service, then keep track of it
	// separately for a quick lookup.
	if s.hasOwnership(service) {
		s.controller.GetC2KContext().SyncedKubeServiceCache[connector.KubeSvcName(service.Name)] = service
		s.controller.GetC2KContext().SyncedKubeServiceHash[connector.KubeSvcName(service.Name)] = s.serviceHash(service)
		s.trigger() // Always trigger sync
	}

	log.Trace().Msgf("UpsertService, key:%s", key)
	return nil
}

// Delete implements the controller.Resource interface.
func (s *CtoKSyncer) Delete(key string, _ interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	service, ok := s.controller.GetC2KContext().KubeServiceCache[connector.KubeSvcKey(key)]
	if !ok {
		// This is a weird scenario, but in unit tests we've seen this happen
		// in cases where the delete happens very quickly after the create.
		// Just to be sure, lets trigger a sync. This is cheap cause it'll
		// do nothing if there is no work to be done.
		s.trigger()
		return nil
	}

	delete(s.controller.GetC2KContext().KubeServiceCache, connector.KubeSvcKey(key))

	kubeSvcName := connector.KubeSvcName(service.Name)
	delete(s.controller.GetC2KContext().SyncedKubeServiceCache, kubeSvcName)
	delete(s.controller.GetC2KContext().SyncedKubeServiceHash, kubeSvcName)

	// If the service that is deleted is part of cloud services, then
	// we need to trigger a sync to recreate it.
	if _, ok = s.controller.GetC2KContext().SourceServices[kubeSvcName]; ok {
		s.trigger()
	}

	log.Info().Msgf("delete service, key:%s name:%s", key, kubeSvcName)
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

	if deriveNamespace, err := s.kubeClient.CoreV1().Namespaces().Get(s.ctx, s.namespace(), metav1.GetOptions{}); err == nil {
		if IsSyncCloudNamespace(deriveNamespace) {
			s.fillEndpoints = false
		} else {
			s.fillEndpoints = true
		}
	}

	svcClient := s.kubeClient.CoreV1().Services(s.namespace())
	eptClient := s.kubeClient.CoreV1().Endpoints(s.namespace())

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
		} else {
			continue
		}

		fromTs := time.Now()
		if len(deletes) > 0 {
			var wg sync.WaitGroup
			wg.Add(len(deletes))
			for _, serviceName := range deletes {
				syncJob := &DeleteSyncJob{
					SyncJob: &SyncJob{
						done: make(chan struct{}),
					},
					ctx:         s.ctx,
					wg:          &wg,
					syncer:      s,
					svcClient:   svcClient,
					serviceName: string(serviceName),
				}
				s.syncWorkQueues.AddJob(syncJob)
			}
			wg.Wait()
		}

		if len(creates) > 0 {
			var wg sync.WaitGroup
			wg.Add(len(creates))
			for _, create := range creates {
				syncJob := &CreateSyncJob{
					SyncJob: &SyncJob{
						done: make(chan struct{}),
					},
					ctx:       s.ctx,
					wg:        &wg,
					syncer:    s,
					svcClient: svcClient,
					eptClient: eptClient,
					create:    create,
				}
				s.syncWorkQueues.AddJob(syncJob)
			}
			wg.Wait()
		}
		toTs := time.Now()
		duration := toTs.Sub(fromTs)
		fmt.Printf("k8s service sync deletes:%4d creates:%4d from %v escape %v\n", len(deletes), len(creates), fromTs, duration)
	}
}

// crudList returns the services to create, update, and delete (respectively).
func (s *CtoKSyncer) crudList() (createSvcs []*syncCreate, deleteSvcs []connector.KubeSvcName) {
	extendServices := make(map[connector.KubeSvcName]connector.CloudSvcName, 0)
	ipFamilyPolicy := corev1.IPFamilyPolicySingleStack
	// Determine what needs to be created or updated
	for kubeSvcName, cloudSvcName := range s.controller.GetC2KContext().SourceServices {
		svcMetaMap := s.microAggregator.Aggregate(s.ctx, kubeSvcName)
		if len(svcMetaMap) == 0 {
			if service, exists := s.controller.GetC2KContext().KubeServiceCache[connector.KubeSvcKey(fmt.Sprintf("%s/%s", s.controller.GetDeriveNamespace(), kubeSvcName))]; exists {
				if s.hasOwnership(service) {
					deleteSvcs = append(deleteSvcs, kubeSvcName)
				}
			}
			continue
		}

		fillEndpoints := s.fillEndpoints

		var serviceSpec *corev1.ServiceSpec
		if externalName := s.controller.GetC2KContext().ExternalServices[cloudSvcName]; len(externalName) > 0 {
			serviceSpec = &corev1.ServiceSpec{
				Type:         corev1.ServiceTypeExternalName,
				ExternalName: string(externalName),
			}
			fillEndpoints = false
		} else {
			serviceSpec = &corev1.ServiceSpec{
				Type:           corev1.ServiceTypeClusterIP,
				ClusterIP:      corev1.ClusterIPNone,
				IPFamilies:     []corev1.IPFamily{corev1.IPv4Protocol},
				IPFamilyPolicy: &ipFamilyPolicy,
			}
		}

		for k8sSvcName, svcMeta := range svcMetaMap {
			if service, exists := s.controller.GetC2KContext().KubeServiceCache[connector.KubeSvcKey(fmt.Sprintf("%s/%s", s.controller.GetDeriveNamespace(), k8sSvcName))]; exists {
				if !s.hasOwnership(service) {
					continue
				}
			}
			if len(svcMeta.Endpoints) == 0 {
				deleteSvcs = append(deleteSvcs, k8sSvcName)
				continue
			}
			if fixedPort := s.controller.GetC2KFixedHTTPServicePort(); fixedPort != nil {
				s.mergeFixedHTTPServiceEndpoints(svcMeta)
				if len(svcMeta.Endpoints) == 0 {
					deleteSvcs = append(deleteSvcs, k8sSvcName)
					continue
				}
			}
			if !strings.EqualFold(string(k8sSvcName), string(kubeSvcName)) {
				extendServices[k8sSvcName] = cloudSvcName
			}

			// If this is an already registered service, then update it
			if svc, ok := s.controller.GetC2KContext().SyncedKubeServiceCache[k8sSvcName]; ok {
				svc.Spec = *serviceSpec
				svc.ObjectMeta.Annotations = map[string]string{
					// Ensure we don't sync the service back to cloud
					connector.AnnotationMeshServiceSync:           string(s.discClient.MicroServiceProvider()),
					connector.AnnotationMeshServiceSyncManagedBy:  s.controller.GetConnectorUID(),
					connector.AnnotationCloudServiceInheritedFrom: string(cloudSvcName),
				}
				if svcMeta.HealthCheck {
					svc.ObjectMeta.Annotations[connector.AnnotationCloudHealthCheckService] = True
					svc.ObjectMeta.Annotations[connector.AnnotationServiceSyncK8sToFgw] = False
					svc.ObjectMeta.Annotations[connector.AnnotationServiceSyncK8sToCloud] = False
				}
				s.fillService(svcMeta, svc, fillEndpoints)
				preHv := s.controller.GetC2KContext().SyncedKubeServiceHash[k8sSvcName]
				if preHv == s.serviceHash(svc) {
					log.Trace().Msgf("service already registered in K8S, not registering, name:%s", k8sSvcName)
					continue
				}
				deleteSvcs = append(deleteSvcs, k8sSvcName)
				continue
			}

			// Register!
			createSvc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:   string(k8sSvcName),
					Labels: map[string]string{constants.CloudSourcedServiceLabel: True},
					Annotations: map[string]string{
						// Ensure we don't sync the service back to Cloud
						connector.AnnotationMeshServiceSync:           string(s.discClient.MicroServiceProvider()),
						connector.AnnotationMeshServiceSyncManagedBy:  s.controller.GetConnectorUID(),
						connector.AnnotationCloudServiceInheritedFrom: string(cloudSvcName),
					},
				},
				Spec: *serviceSpec,
			}
			if svcMeta.HealthCheck {
				createSvc.ObjectMeta.Annotations[connector.AnnotationCloudHealthCheckService] = True
				createSvc.ObjectMeta.Annotations[connector.AnnotationServiceSyncK8sToFgw] = False
				createSvc.ObjectMeta.Annotations[connector.AnnotationServiceSyncK8sToCloud] = False
			}
			endpoints := s.fillService(svcMeta, createSvc, fillEndpoints)
			preHv := s.controller.GetC2KContext().SyncedKubeServiceHash[k8sSvcName]
			if preHv == s.serviceHash(createSvc) {
				log.Debug().Msgf("service already registered in K8S, not registering, name:%s", k8sSvcName)
				continue
			}

			syncCreate := &syncCreate{
				service:   createSvc,
				endpoints: endpoints,
			}
			createSvcs = append(createSvcs, syncCreate)
		}
	}

	if len(extendServices) > 0 {
		for kubeSvcName, cloudSvcName := range extendServices {
			s.controller.GetC2KContext().SourceServices[kubeSvcName] = cloudSvcName
		}
	}

	// Determine what needs to be deleted
	for kubeSvcName := range s.controller.GetC2KContext().SyncedKubeServiceCache {
		if _, ok := s.controller.GetC2KContext().SourceServices[kubeSvcName]; !ok {
			deleteSvcs = append(deleteSvcs, kubeSvcName)
		}
	}

	return createSvcs, deleteSvcs
}

func (s *CtoKSyncer) fillService(svcMeta *connector.MicroSvcMeta, createSvc *corev1.Service, fillEndpoints bool) (endpoints *corev1.Endpoints) {
	var endpointPorts []corev1.EndpointPort
	var endpointAddresses []corev1.EndpointAddress

	for targetPort, appProtocol := range svcMeta.TargetPorts {
		if exists := s.existPort(createSvc, targetPort, appProtocol); !exists {
			specPort := corev1.ServicePort{
				Name:       fmt.Sprintf("%s%d", appProtocol, targetPort),
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(targetPort),
				TargetPort: intstr.FromInt32(int32(targetPort)),
			}
			if len(svcMeta.Ports) > 0 {
				if port, exists := svcMeta.Ports[targetPort]; exists {
					specPort.Port = int32(port)
				}
			}
			if appProtocol == constants.ProtocolHTTP {
				specPort.AppProtocol = &protocolHTTP
				if port := s.controller.GetC2KFixedHTTPServicePort(); port != nil {
					specPort.Port = int32(*port)
				}
			}
			if appProtocol == constants.ProtocolGRPC {
				specPort.AppProtocol = &protocolGRPC
			}
			createSvc.Spec.Ports = append(createSvc.Spec.Ports, specPort)
		}

		if fillEndpoints {
			endpointPorts = append(endpointPorts, corev1.EndpointPort{Port: int32(targetPort)})
		}
	}
	for addr, endpointMeta := range svcMeta.Endpoints {
		endpointMeta.Init(s.controller, s.discClient)
		if fillEndpoints {
			endpointAddresses = append(endpointAddresses, corev1.EndpointAddress{IP: string(addr)})
		}
	}

	enc, hash := connector.Encode(svcMeta)
	createSvc.ObjectMeta.Annotations[connector.AnnotationMeshEndpointAddr] = enc
	createSvc.ObjectMeta.Annotations[constants.AnnotationMeshEndpointHash] = fmt.Sprintf("%d", hash)
	if svcMeta.GRPCMeta != nil && len(svcMeta.GRPCMeta.Interface) > 0 {
		createSvc.ObjectMeta.Labels[constants.GRPCServiceInterfaceLabel] = svcMeta.GRPCMeta.Interface
	}

	if fillEndpoints && len(endpointAddresses) > 0 && len(endpointPorts) > 0 {
		endpoints = &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:   createSvc.Name,
				Labels: createSvc.Labels,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: endpointAddresses,
					Ports:     endpointPorts,
				},
			},
		}
	}

	return endpoints
}

func (s *CtoKSyncer) existPort(svc *corev1.Service, port connector.MicroServicePort, appProtocol connector.MicroServiceProtocol) bool {
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

func (s *CtoKSyncer) serviceHash(service *corev1.Service) uint64 {
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
	if len(service.Spec.ExternalName) > 0 {
		externalNameBytes, _ := json.Marshal(service.Spec.ExternalName)
		bytes = append(bytes, externalNameBytes...)
	}
	return utils.Hash(bytes)
}

func (s *CtoKSyncer) mergeFixedHTTPServiceEndpoints(meta *connector.MicroSvcMeta) {
	stats := make(map[connector.MicroServicePort]map[connector.MicroServiceAddress]*connector.MicroEndpointMeta)
	for addr, ep := range meta.Endpoints {
		for port, protocol := range ep.Ports {
			if strings.EqualFold(string(protocol), constants.ProtocolHTTP) {
				epsCache, exists := stats[port]
				if !exists {
					epsCache = make(map[connector.MicroServiceAddress]*connector.MicroEndpointMeta)
					stats[port] = epsCache
				}
				epsCache[addr] = ep
			}
		}
	}

	var peak = 0
	var targetPorts map[connector.MicroServicePort]int
	for port, epsCache := range stats {
		cnt := len(epsCache)
		if cnt == 0 {
			continue
		} else if cnt == peak {
			targetPorts[port] = cnt
		} else if cnt > peak {
			peak = cnt
			targetPorts = make(map[connector.MicroServicePort]int)
			targetPorts[port] = cnt
		}
	}

	if len(targetPorts) > 0 {
		valley := connector.MicroServicePort(0)
		for port := range targetPorts {
			if port > valley {
				valley = port
			}
		}
		meta.TargetPorts = make(map[connector.MicroServicePort]connector.MicroServiceProtocol)
		meta.TargetPorts[valley] = connector.ProtocolHTTP
		meta.Endpoints = stats[valley]
	}
}
