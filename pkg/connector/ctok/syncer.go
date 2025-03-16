package ctok

import (
	"context"
	"encoding/json"
	"fmt"
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

func (s *CtoKSyncer) toLegalServiceName(serviceName string) string {
	serviceName = strings.ReplaceAll(serviceName, "_", "-")
	serviceName = strings.ReplaceAll(serviceName, ".", "-")
	serviceName = strings.ReplaceAll(serviceName, " ", "-")
	serviceName = strings.ToLower(serviceName)
	return serviceName
}

func (s *CtoKSyncer) SetMicroAggregator(microAggregator Aggregator) {
	s.microAggregator = microAggregator
}

// SetServices is called with the services that should be created.
// The key is the service name and the destination is the external DNS
// entry to point to.
func (s *CtoKSyncer) SetServices(svcs map[connector.MicroSvcName]connector.MicroSvcDomainName) {
	s.lock.Lock()
	defer s.lock.Unlock()

	legalSvcNames := make(map[string]string)
	for serviceName := range svcs {
		legalSvcNames[s.toLegalServiceName(string(serviceName))] = string(serviceName)
	}

	s.controller.GetC2KContext().SourceServices = legalSvcNames
	s.controller.GetC2KContext().RawServices = maps.Clone(legalSvcNames)
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
			&apiv1.Service{},
			0,
			cache.Indexers{},
		)
	}
	return s.informer
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
	if service.Labels != nil && service.Labels[constants.CloudSourcedServiceLabel] == True {
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

	svcClient := s.kubeClient.CoreV1().Services(s.namespace())

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
					serviceName: serviceName,
				}
				s.syncWorkQueues.AddJob(syncJob)
			}
			wg.Wait()
		}

		if len(creates) > 0 {
			var wg sync.WaitGroup
			wg.Add(len(creates))
			for _, service := range creates {
				syncJob := &CreateSyncJob{
					SyncJob: &SyncJob{
						done: make(chan struct{}),
					},
					ctx:       s.ctx,
					wg:        &wg,
					syncer:    s,
					svcClient: svcClient,
					service:   *service,
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
func (s *CtoKSyncer) crudList() ([]*apiv1.Service, []string) {
	var createSvcs []*apiv1.Service
	var deleteSvcs []string
	extendServices := make(map[string]string, 0)
	ipFamilyPolicy := apiv1.IPFamilyPolicySingleStack
	// Determine what needs to be created or updated
	for k8sSvcName, cloudSvcName := range s.controller.GetC2KContext().SourceServices {
		svcMetaMap, labels, annotations, err := s.microAggregator.Aggregate(s.ctx, connector.MicroSvcName(k8sSvcName))
		if err != nil {
			log.Warn().Err(err).Msg("fail to get service instances")
			continue
		}
		if len(svcMetaMap) == 0 {
			if _, exists := s.controller.GetC2KContext().ServiceKeyToName[fmt.Sprintf("%s/%s", s.controller.GetDeriveNamespace(), k8sSvcName)]; exists {
				deleteSvcs = append(deleteSvcs, k8sSvcName)
			}
			continue
		}

		labels[constants.CloudSourcedServiceLabel] = True
		annotations[connector.AnnotationMeshServiceSync] = string(s.discClient.MicroServiceProvider())
		annotations[connector.AnnotationCloudServiceInheritedFrom] = cloudSvcName

		for microSvcName, svcMeta := range svcMetaMap {
			if len(svcMeta.Endpoints) == 0 {
				deleteSvcs = append(deleteSvcs, string(microSvcName))
				continue
			}
			if fixedPort := s.controller.GetC2KFixedHTTPServicePort(); fixedPort != nil {
				s.mergeFixedHTTPServiceEndpoints(svcMeta)
				if len(svcMeta.Endpoints) == 0 {
					deleteSvcs = append(deleteSvcs, string(microSvcName))
					continue
				}
			}
			if fixedPort := s.controller.GetC2KFixedGRPCServicePort(); fixedPort != nil {
				s.mergeFixedGRPCServiceEndpoints(svcMeta)
				if len(svcMeta.Endpoints) == 0 {
					deleteSvcs = append(deleteSvcs, string(microSvcName))
					continue
				}
			}
			if !strings.EqualFold(string(microSvcName), k8sSvcName) {
				extendServices[string(microSvcName)] = cloudSvcName
			}

			// If this is an already registered service, then update it
			if svc, ok := s.controller.GetC2KContext().ServiceMapCache[string(microSvcName)]; ok {
				svc.ObjectMeta.Labels = maps.Clone(labels)
				svc.ObjectMeta.Annotations = maps.Clone(annotations)
				if svcMeta.HealthCheck {
					svc.Annotations[connector.AnnotationCloudHealthCheckService] = True
					svc.Annotations[connector.AnnotationServiceSyncK8sToFgw] = False
					svc.Annotations[connector.AnnotationServiceSyncK8sToCloud] = False
				}
				s.fillService(svcMeta, svc)
				preHv := s.controller.GetC2KContext().ServiceHashMap[string(microSvcName)]
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
					Name:        string(microSvcName),
					Labels:      maps.Clone(labels),
					Annotations: maps.Clone(annotations),
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
			if svcMeta.HealthCheck {
				createSvc.Annotations[connector.AnnotationCloudHealthCheckService] = True
				createSvc.Annotations[connector.AnnotationServiceSyncK8sToFgw] = False
				createSvc.Annotations[connector.AnnotationServiceSyncK8sToCloud] = False
			}
			s.fillService(svcMeta, createSvc)
			preHv := s.controller.GetC2KContext().ServiceHashMap[string(microSvcName)]
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

func (s *CtoKSyncer) fillService(svcMeta *connector.MicroSvcMeta, createSvc *apiv1.Service) {
	for targetPort, appProtocol := range svcMeta.Ports {
		if exists := s.existPort(createSvc, targetPort, appProtocol); !exists {
			specPort := apiv1.ServicePort{
				Name:       fmt.Sprintf("%s%d", appProtocol, targetPort),
				Protocol:   apiv1.ProtocolTCP,
				Port:       int32(targetPort),
				TargetPort: intstr.FromInt32(int32(targetPort)),
			}
			if appProtocol == constants.ProtocolHTTP {
				specPort.AppProtocol = &protocolHTTP
				if port := s.controller.GetC2KFixedHTTPServicePort(); port != nil {
					specPort.Port = int32(*port)
				}
			}
			if appProtocol == constants.ProtocolGRPC {
				specPort.AppProtocol = &protocolGRPC
				if port := s.controller.GetC2KFixedGRPCServicePort(); port != nil {
					specPort.Port = int32(*port)
				}
			}
			createSvc.Spec.Ports = append(createSvc.Spec.Ports, specPort)
		}
	}
	for _, endpointMeta := range svcMeta.Endpoints {
		endpointMeta.Init(s.controller, s.discClient)
	}

	enc, hash := connector.Encode(svcMeta)
	createSvc.ObjectMeta.Annotations[connector.AnnotationMeshEndpointAddr] = enc
	createSvc.ObjectMeta.Annotations[constants.AnnotationMeshEndpointHash] = fmt.Sprintf("%d", hash)
}

func (s *CtoKSyncer) existPort(svc *apiv1.Service, port connector.MicroSvcPort, appProtocol connector.MicroSvcAppProtocol) bool {
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

func (s *CtoKSyncer) mergeFixedHTTPServiceEndpoints(meta *connector.MicroSvcMeta) {
	stats := make(map[connector.MicroSvcPort]map[connector.MicroEndpointAddr]*connector.MicroEndpointMeta)
	for addr, ep := range meta.Endpoints {
		for port, protocol := range ep.Ports {
			if strings.EqualFold(string(protocol), constants.ProtocolHTTP) {
				epsCache, exists := stats[port]
				if !exists {
					epsCache = make(map[connector.MicroEndpointAddr]*connector.MicroEndpointMeta)
					stats[port] = epsCache
				}
				epsCache[addr] = ep
			}
		}
	}

	var peak = 0
	var targetPorts map[connector.MicroSvcPort]int
	for port, epsCache := range stats {
		cnt := len(epsCache)
		if cnt == 0 {
			continue
		} else if cnt == peak {
			targetPorts[port] = cnt
		} else if cnt > peak {
			peak = cnt
			targetPorts = make(map[connector.MicroSvcPort]int)
			targetPorts[port] = cnt
		}
	}

	if len(targetPorts) > 0 {
		valley := connector.MicroSvcPort(0)
		for port := range targetPorts {
			if port > valley {
				valley = port
			}
		}
		meta.Ports = make(map[connector.MicroSvcPort]connector.MicroSvcAppProtocol)
		meta.Ports[valley] = constants.ProtocolHTTP
		meta.Endpoints = stats[valley]
	}
}

func (s *CtoKSyncer) mergeFixedGRPCServiceEndpoints(meta *connector.MicroSvcMeta) {
	stats := make(map[connector.MicroSvcPort]map[connector.MicroEndpointAddr]*connector.MicroEndpointMeta)
	for addr, ep := range meta.Endpoints {
		for port, protocol := range ep.Ports {
			if strings.EqualFold(string(protocol), constants.ProtocolGRPC) {
				epsCache, exists := stats[port]
				if !exists {
					epsCache = make(map[connector.MicroEndpointAddr]*connector.MicroEndpointMeta)
					stats[port] = epsCache
				}
				epsCache[addr] = ep
			}
		}
	}

	var peak = 0
	var targetPorts map[connector.MicroSvcPort]int
	for port, epsCache := range stats {
		cnt := len(epsCache)
		if cnt == 0 {
			continue
		} else if cnt == peak {
			targetPorts[port] = cnt
		} else if cnt > peak {
			peak = cnt
			targetPorts = make(map[connector.MicroSvcPort]int)
			targetPorts[port] = cnt
		}
	}

	if len(targetPorts) > 0 {
		valley := connector.MicroSvcPort(0)
		for port := range targetPorts {
			if port > valley {
				valley = port
			}
		}
		meta.Ports = make(map[connector.MicroSvcPort]connector.MicroSvcAppProtocol)
		meta.Ports[valley] = constants.ProtocolGRPC
		meta.Endpoints = stats[valley]
	}
}
