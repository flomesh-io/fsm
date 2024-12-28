package ktoc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/announcements"
	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

var (
	log = logger.New("connector-k2c")

	ProtocolHTTP = constants.ProtocolHTTP
	ProtocolGRPC = constants.ProtocolGRPC
)

const (
	// cloudKubernetesCheckType is the type of health check in cloud for Kubernetes readiness status.
	cloudKubernetesCheckType = "kubernetes-readiness"
	// cloudKubernetesCheckName is the name of health check in cloud for Kubernetes readiness status.
	cloudKubernetesCheckName   = "Kubernetes Readiness Check"
	kubernetesSuccessReasonMsg = "Kubernetes health checks passing"
)

// KtoCSource implements controller.Resource to sync RegisteredInstances resource
// types from K8S.
type KtoCSource struct {
	controller connector.ConnectController
	syncer     Syncer
	discClient connector.ServiceDiscoveryClient

	msgBroker     *messaging.Broker
	msgWorkQueues *workerpool.WorkerPool

	// serviceLock must be held for any read/write to these maps.
	serviceLock  sync.Mutex
	serviceDetas uint64

	kubeClient kubernetes.Interface

	endpointsResource *serviceEndpointsSource

	// ctx is used to cancel processes kicked off by KtoCSource.
	ctx context.Context
}

func NewKtoCSource(controller connector.ConnectController,
	syncer Syncer,
	ctx context.Context,
	msgBroker *messaging.Broker,
	kubeClient kubernetes.Interface,
	discClient connector.ServiceDiscoveryClient) *KtoCSource {
	return &KtoCSource{
		controller:    controller,
		syncer:        syncer,
		discClient:    discClient,
		msgBroker:     msgBroker,
		msgWorkQueues: workerpool.NewWorkerPool(1),
		kubeClient:    kubeClient,
		ctx:           ctx,
	}
}

func (t *KtoCSource) Lock() {
	t.serviceLock.Lock()
}

func (t *KtoCSource) Unlock() {
	t.serviceLock.Unlock()
}

// Informer implements the controller.Resource interface.
func (t *KtoCSource) Informer() cache.SharedIndexInformer {
	// Watch all k8s namespaces. Events will be filtered out as appropriate
	// based on the allow and deny lists in the `shouldSync` function.
	return cache.NewSharedIndexInformer(
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

// Upsert implements the controller.Resource interface.
func (t *KtoCSource) Upsert(key string, raw interface{}) error {
	// We expect a RegisteredInstances. If it isn't a service then just ignore it.
	svc, ok := raw.(*corev1.Service)
	if !ok {
		log.Warn().Msgf("upsert got invalid type raw:%v", raw)
		return nil
	}

	t.Lock()
	defer t.Unlock()

	if !t.shouldSync(svc) {
		// Check if its in our map and delete it.
		if _, ok = t.controller.GetK2CContext().ServiceMap.Get(key); ok {
			log.Info().Msgf("service should no longer be synced service:%s", key)
			t.doDelete(key)
		} else {
			log.Debug().Msgf("[KtoCSource.Upsert] syncing disabled for service, ignoring key:%s", key)
		}
		return nil
	}

	// Syncing is enabled, let's keep track of this service.
	t.controller.GetK2CContext().ServiceMap.Set(key, svc)
	log.Debug().Msgf("[KtoCSource.Upsert] adding service to serviceMap key:%s service:%v", key, svc)

	// If we care about endpoints, we should do the initial endpoints load.
	if t.shouldTrackEndpoints(key) {
		cloudService := false
		if len(svc.Annotations) > 0 {
			if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
				cloudService = true
				svcMeta := connector.Decode(svc, v)
				endpoints := new(corev1.Endpoints)
				endpointSubset := corev1.EndpointSubset{}
				for port, protocol := range svcMeta.Ports {
					endpointPort := corev1.EndpointPort{}
					endpointPort.Port = int32(port)
					endpointPort.Protocol = constants.ProtocolTCP
					if protocol == connector.ProtocolHTTP {
						endpointPort.AppProtocol = &ProtocolHTTP
					} else if protocol == connector.ProtocolGRPC {
						endpointPort.AppProtocol = &ProtocolGRPC
					}
					endpointSubset.Ports = append(endpointSubset.Ports, endpointPort)
				}
				for ip := range svcMeta.Endpoints {
					endpointAddress := corev1.EndpointAddress{}
					endpointAddress.IP = string(ip)
					endpointSubset.Addresses = append(endpointSubset.Addresses, endpointAddress)
				}
				endpoints.Subsets = append(endpoints.Subsets, endpointSubset)
				t.controller.GetK2CContext().EndpointsMap.Set(key, endpoints)
				log.Debug().Msgf("[KtoCSource.Upsert] adding service's endpoints to endpointsMap key:%s service:%v endpoints:%v", key, svc, endpoints)
			}
		}
		if !cloudService {
			endpoints, err := t.endpointsResource.getEndpoints(key)
			if err != nil {
				log.Debug().Msgf("error loading initial endpoints key%s err:%v",
					key,
					err)
			} else {
				t.controller.GetK2CContext().EndpointsMap.Set(key, endpoints)
				log.Debug().Msgf("[KtoCSource.Upsert] adding service's endpoints to endpointsMap key:%s service:%v endpoints:%v", key, svc, endpoints)
			}
		}
	}

	// Update the registration and trigger a sync
	t.generateRegistrations(key)
	t.sync()
	log.Info().Msgf("upsert key:%s", key)
	return nil
}

// Delete implements the controller.Resource interface.
func (t *KtoCSource) Delete(key string, _ interface{}) error {
	t.Lock()
	defer t.Unlock()
	t.doDelete(key)
	log.Info().Msgf("delete key:%s", key)
	return nil
}

// doDelete is a helper function for deletion.
//
// Precondition: assumes t.serviceLock is held.
func (t *KtoCSource) doDelete(key string) {
	t.controller.GetK2CContext().ServiceMap.Remove(key)
	log.Debug().Msgf("[doDelete] deleting service from serviceMap key:%s", key)
	t.controller.GetK2CContext().EndpointsMap.Remove(key)
	log.Debug().Msgf("[doDelete] deleting endpoints from endpointsMap key:%s", key)
	// If there were registrations related to this service, then
	// delete them and sync.
	t.controller.GetK2CContext().RegisteredServiceMap.Remove(key)
	t.sync()
}

// Run implements the controller.Backgrounder interface.
func (t *KtoCSource) Run(ch <-chan struct{}) {
	log.Info().Msg("starting runner for endpoints")
	// Register a controller for Endpoints which subsequently registers a
	// controller for the Ingress resource.
	t.endpointsResource = &serviceEndpointsSource{
		Service: t,
		Ctx:     t.ctx,
		Resource: &serviceIngressSource{
			Service: t,
			Ctx:     t.ctx,
		},
	}
	(&connector.CacheController{
		Resource: t.endpointsResource,
	}).Run(ch)
}

// shouldSync returns true if resyncing should be enabled for the given service.
func (t *KtoCSource) shouldSync(svc *corev1.Service) bool {
	// Namespace logic
	if deriveNamespace := t.controller.GetDeriveNamespace(); strings.EqualFold(svc.Namespace, deriveNamespace) {
		log.Debug().Msgf("[shouldSync] service is in the deny list svc.Namespace:%s service:%v", svc.Namespace, svc)
		return false
	}

	// If in deny list, don't sync
	if t.controller.GetDenyK8SNamespaceSet().Contains(svc.Namespace) {
		log.Debug().Msgf("[shouldSync] service is in the deny list svc.Namespace:%s service:%v", svc.Namespace, svc)
		return false
	}

	// If not in allow list or allow list is not *, don't sync
	if !t.controller.GetAllowK8SNamespaceSet().Contains("*") && !t.controller.GetAllowK8SNamespaceSet().Contains(svc.Namespace) {
		log.Debug().Msgf("[shouldSync] service not in allow list svc.Namespace:%s service:%v", svc.Namespace, svc)
		return false
	}

	// Ignore ClusterIP services if ClusterIP sync is disabled
	if svc.Spec.Type == corev1.ServiceTypeClusterIP && !t.controller.GetSyncClusterIPServices() {
		log.Debug().Msgf("[shouldSync] ignoring clusterip service svc.Namespace:%s service:%v", svc.Namespace, svc)
		return false
	}

	if len(svc.Annotations) > 0 {
		hasLocalInstance := false
		if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
			svcMeta := connector.Decode(svc, v)
			for _, endpointMeta := range svcMeta.Endpoints {
				if endpointMeta.Local.InternalService {
					hasLocalInstance = true
					break
				}
			}
		} else {
			hasLocalInstance = true
		}
		if !hasLocalInstance {
			return false
		}
	}

	raw, ok := svc.Annotations[connector.AnnotationServiceSyncK8sToCloud]
	if !ok {
		// If there is no explicit value, then set it to our current default.
		return t.controller.GetDefaultSync()
	}

	v, err := strconv.ParseBool(raw)
	if err != nil {
		log.Warn().Msgf("error parsing service-sync annotation service-name:%s err:%v",
			t.addPrefixAndK8SNamespace(svc.Name, svc.Namespace),
			err)

		// Fallback to default
		return t.controller.GetDefaultSync()
	}

	return v
}

// shouldTrackEndpoints returns true if the endpoints for the given key
// should be tracked.
//
// Precondition: this requires the lock to be held.
func (t *KtoCSource) shouldTrackEndpoints(key string) bool {
	// The service must be one we care about for us to watch the endpoints.
	// We care about a service that exists in our service map (is enabled
	// for syncing) and is a NodePort or ClusterIP type since only those
	// types use endpoints.
	if t.controller.GetK2CContext().ServiceMap.IsEmpty() {
		return false
	}
	svc, ok := t.controller.GetK2CContext().ServiceMap.Get(key)
	if !ok {
		return false
	}

	return svc.Spec.Type == corev1.ServiceTypeNodePort ||
		svc.Spec.Type == corev1.ServiceTypeClusterIP ||
		(t.controller.GetSyncLoadBalancerEndpoints() && svc.Spec.Type == corev1.ServiceTypeLoadBalancer)
}

// generateRegistrations generates the necessary cloud registrations for
// the given key. This is best effort: if there isn't enough information
// yet to register a service, then no registration will be generated.
//
// Precondition: the lock t.lock is held.
func (t *KtoCSource) generateRegistrations(key string) {
	// Get the service. If it doesn't exist, then we can't generate.
	svc, ok := t.controller.GetK2CContext().ServiceMap.Get(key)
	if !ok {
		return
	}

	log.Debug().Msgf("[generateRegistrations] generating registration key:%s", key)

	// Begin by always clearing the old value out since we'll regenerate
	// a new one if there is one.
	t.controller.GetK2CContext().RegisteredServiceMap.Remove(key)

	var svcMeta *connector.MicroSvcMeta
	if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
		svcMeta = connector.Decode(svc, v)
	}

	// baseNode and baseService are the base that should be modified with
	// service-type specific changes. These are not pointers, they should be
	// shallow copied for each instance.
	baseNode := connector.CatalogRegistration{
		SkipNodeUpdate: true,
		NodeMeta: map[string]string{
			connector.ClusterSetKey: t.controller.GetClusterSet(),
		},
	}

	if t.controller.GetK2CWithGateway() {
		if ingressAddr := t.controller.GetViaIngressAddr(); len(ingressAddr) > 0 {
			baseNode.Address = ingressAddr
		}
	}

	baseService := connector.AgentService{
		MicroService: connector.MicroService{
			NamespacedService: connector.NamespacedService{
				Service: t.addPrefixAndK8SNamespace(svc.Name, svc.Namespace),
			},
		},
		Meta: map[string]interface{}{
			connector.ClusterSetKey: t.controller.GetClusterSet(),
			connector.ConnectUIDKey: t.controller.GetConnectorUID(),
			connector.CloudK8SNS:    svc.Namespace,
		},
	}

	// If the name is explicitly annotated, adopt that name
	if v, ok := svc.Annotations[connector.AnnotationServiceName]; ok {
		baseService.MicroService.Service = strings.TrimSpace(v)
	} else if v, ok := svc.Annotations[connector.AnnotationCloudServiceInheritedFrom]; ok {
		baseService.MicroService.Service = strings.TrimSpace(v)
	}

	// Update the service namespace based on namespace settings
	registeredNS := t.discClient.RegisteredNamespace(svc.Namespace)
	if registeredNS != "" {
		log.Debug().Msgf("[generateRegistrations] namespace being used key:%s namespace:%s", key, registeredNS)
		baseService.MicroService.Namespace = registeredNS
	}

	// Determine the default port and set port annotations
	overridePortName, overridePortNumber := t.determinePortAnnotations(svc, baseService)

	// Parse any additional tags
	if rawTags, ok := svc.Annotations[connector.AnnotationServiceTags]; ok {
		baseService.Tags = append(baseService.Tags, parseTags(rawTags)...)
	}

	// Parse any additional meta
	for k, v := range svc.Annotations {
		if strings.HasPrefix(k, connector.AnnotationServiceMetaPrefix) {
			k = strings.TrimPrefix(k, connector.AnnotationServiceMetaPrefix)
			baseService.Meta[k] = v
		}
	}

	// Always log what we generated
	defer func() {
		log.Debug().Msgf("generated registration key:%s service:%s namespace:%s instances:%d",
			key,
			baseService.MicroService.Service,
			baseService.MicroService.Namespace,
			t.controller.GetK2CContext().RegisteredServiceMap.Count())
	}()

	// If there are external IPs then those become the instance registrations
	// for any type of service.
	if t.generateExternalIPRegistrations(key, svc, baseNode, baseService) {
		return
	}

	switch svc.Spec.Type {
	// For LoadBalancer type services, we create a service instance for
	// each LoadBalancer entry. We only support entries that have an IP
	// address assigned (not hostnames).
	// If loadBalancerEndpointsSync is true sync LB endpoints instead of loadbalancer ingress.
	case corev1.ServiceTypeLoadBalancer:
		t.generateLoadBalanceEndpointsRegistrations(svcMeta, key, baseNode, baseService, overridePortName, overridePortNumber, svc)

	// For NodePort services, we create a service instance for each
	// endpoint of the service, which corresponds to the nodes the service's
	// pods are running on. This way we don't register _every_ K8S
	// node as part of the service.
	case corev1.ServiceTypeNodePort:
		if t.generateNodeportRegistrations(key, baseNode, baseService) {
			return
		}

	// For ClusterIP services, we register a service instance
	// for each endpoint.
	case corev1.ServiceTypeClusterIP:
		t.registerServiceInstance(svcMeta, baseNode, baseService, key, overridePortName, overridePortNumber, true)
	}
}

func (t *KtoCSource) determinePortAnnotations(svc *corev1.Service, baseService connector.AgentService) (string, int32) {
	var overridePortName string
	var overridePortNumber int32
	if len(svc.Spec.Ports) > 0 {
		var port int32
		isNodePort := svc.Spec.Type == corev1.ServiceTypeNodePort

		// If a specific port is specified, then use that port value
		portAnnotation, ok := svc.Annotations[connector.AnnotationServicePort]
		if ok {
			if v, err := strconv.ParseInt(portAnnotation, 0, 0); err == nil {
				port = int32(v)
				overridePortNumber = port
			} else {
				overridePortName = portAnnotation
			}
		}

		// For when the port was a name instead of an int
		if overridePortName != "" {
			// Find the named port
			for _, p := range svc.Spec.Ports {
				if p.Name == overridePortName {
					if isNodePort && p.NodePort > 0 {
						port = p.NodePort
					} else {
						port = p.Port
						// NOTE: for cluster IP services we always use the endpoint
						// ports so this will be overridden.
					}
					break
				}
			}
		}

		// If the port was not set above, set it with the first port
		// based on the service type.
		if port == 0 {
			if isNodePort {
				// Find first defined NodePort
				for _, p := range svc.Spec.Ports {
					if p.NodePort > 0 {
						port = p.NodePort
						break
					}
				}
			} else {
				port = svc.Spec.Ports[0].Port
				// NOTE: for cluster IP services we always use the endpoint
				// ports so this will be overridden.
			}
		}

		baseService.MicroService.EndpointPort().Set(port)

		// Add all the ports as annotations
		for _, p := range svc.Spec.Ports {
			// Set the tag
			baseService.Meta[connector.CloudK8SPort+"-"+p.Name] = strconv.FormatInt(int64(p.Port), 10)
		}
	}
	return overridePortName, overridePortNumber
}

func (t *KtoCSource) generateExternalIPRegistrations(key string, svc *corev1.Service, baseNode connector.CatalogRegistration, baseService connector.AgentService) bool {
	if ips := svc.Spec.ExternalIPs; len(ips) > 0 {
		for _, ip := range ips {
			r := baseNode
			rs := baseService
			r.Service = &rs
			r.Service.ID = t.controller.GetServiceInstanceID(r.Service.MicroService.Service, ip, *rs.MicroService.EndpointPort(), *rs.MicroService.Protocol())
			r.Service.MicroService.EndpointAddress().Set(ip)
			// Adding information about service weight.
			// Overrides the existing weight if present.
			if weight, ok := svc.Annotations[connector.AnnotationServiceWeight]; ok && weight != "" {
				weightI, err := getServiceWeight(weight)
				if err == nil {
					r.Service.Weights = connector.AgentWeights{
						Passing: weightI,
					}
				} else {
					log.Debug().Msgf("[generateRegistrations] service weight err:%v", err)
				}
			}

			t.controller.GetK2CContext().RegisteredServiceMap.Upsert(
				key,
				[]*connector.CatalogRegistration{&r},
				func(exist bool,
					valueInMap []*connector.CatalogRegistration,
					newValue []*connector.CatalogRegistration,
				) []*connector.CatalogRegistration {
					return append(valueInMap, newValue...)
				})
		}

		return true
	}
	return false
}

func (t *KtoCSource) generateNodeportRegistrations(key string, baseNode connector.CatalogRegistration, baseService connector.AgentService) bool {
	if t.controller.GetK2CContext().EndpointsMap.IsEmpty() {
		return true
	}

	endpoints, ok := t.controller.GetK2CContext().EndpointsMap.Get(key)
	if !ok || endpoints == nil || len(endpoints.Subsets) == 0 {
		return true
	}

	for _, subset := range endpoints.Subsets {
		for _, subsetAddr := range subset.Addresses {
			// Check that the node name exists
			// subsetAddr.NodeName is of type *string
			if subsetAddr.NodeName == nil {
				continue
			}

			// Look up the node's ip address by getting node info
			node, err := t.kubeClient.CoreV1().Nodes().Get(t.ctx, *subsetAddr.NodeName, metav1.GetOptions{})
			if err != nil {
				log.Warn().Msgf("error getting node info error:%v", err)
				continue
			}

			// Set the expected node address type
			var expectedType corev1.NodeAddressType
			if t.controller.GetNodePortSyncType() == ctv1.InternalOnly {
				expectedType = corev1.NodeInternalIP
			} else {
				expectedType = corev1.NodeExternalIP
			}

			// Find the ip address for the node and
			// create the cloud service using it
			var found bool
			for _, address := range node.Status.Addresses {
				if address.Type == expectedType {
					if !t.filterIPRanges(address.Address) || t.excludeIPRanges(address.Address) {
						continue
					}

					found = true
					r := baseNode
					rs := baseService
					r.Service = &rs
					r.Service.ID = t.controller.GetServiceInstanceID(r.Service.MicroService.Service, subsetAddr.IP, *rs.MicroService.EndpointPort(), *rs.MicroService.Protocol())
					r.Service.MicroService.EndpointAddress().Set(address.Address)

					t.controller.GetK2CContext().RegisteredServiceMap.Upsert(
						key,
						[]*connector.CatalogRegistration{&r},
						func(exist bool,
							valueInMap []*connector.CatalogRegistration,
							newValue []*connector.CatalogRegistration,
						) []*connector.CatalogRegistration {
							return append(valueInMap, newValue...)
						})
					// Only consider the first address that matches. In some cases
					// there will be multiple addresses like when using AWS CNI.
					// In those cases, Kubernetes will ensure eth0 is always the first
					// address in the list.
					// See https://github.com/kubernetes/kubernetes/blob/b559434c02f903dbcd46ee7d6c78b216d3f0aca0/staging/src/k8s.io/legacy-cloud-providers/aws/aws.go#L1462-L1464
					break
				}
			}

			// If an ExternalIP wasn't found, and ExternalFirst is set,
			// use an InternalIP
			if t.controller.GetNodePortSyncType() == ctv1.ExternalFirst && !found {
				for _, address := range node.Status.Addresses {
					if address.Type == corev1.NodeInternalIP {
						if !t.filterIPRanges(address.Address) {
							continue
						}

						if t.excludeIPRanges(address.Address) {
							continue
						}

						r := baseNode
						rs := baseService
						r.Service = &rs
						r.Service.ID = t.controller.GetServiceInstanceID(r.Service.MicroService.Service, subsetAddr.IP, *rs.MicroService.EndpointPort(), *rs.MicroService.Protocol())
						r.Service.MicroService.EndpointAddress().Set(address.Address)

						t.controller.GetK2CContext().RegisteredServiceMap.Upsert(
							key,
							[]*connector.CatalogRegistration{&r},
							func(exist bool,
								valueInMap []*connector.CatalogRegistration,
								newValue []*connector.CatalogRegistration,
							) []*connector.CatalogRegistration {
								return append(valueInMap, newValue...)
							})
						// Only consider the first address that matches. In some cases
						// there will be multiple addresses like when using AWS CNI.
						// In those cases, Kubernetes will ensure eth0 is always the first
						// address in the list.
						// See https://github.com/kubernetes/kubernetes/blob/b559434c02f903dbcd46ee7d6c78b216d3f0aca0/staging/src/k8s.io/legacy-cloud-providers/aws/aws.go#L1462-L1464
						break
					}
				}
			}
		}
	}
	return false
}

func (t *KtoCSource) generateLoadBalanceEndpointsRegistrations(
	svcMeta *connector.MicroSvcMeta,
	key string,
	baseNode connector.CatalogRegistration,
	baseService connector.AgentService,
	overridePortName string,
	overridePortNumber int32,
	svc *corev1.Service) {
	if t.controller.GetSyncLoadBalancerEndpoints() {
		t.registerServiceInstance(svcMeta, baseNode, baseService, key, overridePortName, overridePortNumber, false)
	} else {
		seen := map[string]struct{}{}
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			addr := ingress.IP
			if addr == "" {
				addr = ingress.Hostname
			}
			if addr == "" {
				continue
			}

			if !t.filterIPRanges(addr) || t.excludeIPRanges(addr) {
				continue
			}

			if _, ok := seen[addr]; ok {
				continue
			}
			seen[addr] = struct{}{}

			r := baseNode
			rs := baseService

			if len(overridePortName) > 0 {
				for _, p := range svc.Spec.Ports {
					if overridePortName == p.Name {
						rs.MicroService.SetHTTPPort(p.Port)
						break
					}
				}
			} else if overridePortNumber > 0 {
				rs.MicroService.SetHTTPPort(overridePortNumber)
			}
			r.Service = &rs
			r.Service.ID = t.controller.GetServiceInstanceID(r.Service.MicroService.Service, addr, *rs.MicroService.EndpointPort(), *rs.MicroService.Protocol())
			r.Service.MicroService.EndpointAddress().Set(addr)

			// Adding information about service weight.
			// Overrides the existing weight if present.
			if weight, ok := svc.Annotations[connector.AnnotationServiceWeight]; ok && weight != "" {
				weightI, err := getServiceWeight(weight)
				if err == nil {
					r.Service.Weights = connector.AgentWeights{
						Passing: weightI,
					}
				} else {
					log.Debug().Msgf("[generateRegistrations] service weight err:%v", err)
				}
			}

			t.controller.GetK2CContext().RegisteredServiceMap.Upsert(
				key,
				[]*connector.CatalogRegistration{&r},
				func(exist bool,
					valueInMap []*connector.CatalogRegistration,
					newValue []*connector.CatalogRegistration,
				) []*connector.CatalogRegistration {
					return append(valueInMap, newValue...)
				})
		}
	}
}

func (t *KtoCSource) registerServiceInstance(
	svcMeta *connector.MicroSvcMeta,
	baseNode connector.CatalogRegistration,
	baseService connector.AgentService,
	key string,
	overridePortName string,
	overridePortNumber int32,
	useHostname bool) {
	if t.controller.GetK2CContext().EndpointsMap.IsEmpty() {
		return
	}

	endpoints, ok := t.controller.GetK2CContext().EndpointsMap.Get(key)
	if !ok || endpoints == nil || len(endpoints.Subsets) == 0 {
		return
	}

	seen := map[connector.MicroServiceAddress]struct{}{}
	for _, subset := range endpoints.Subsets {
		// For ClusterIP services and if loadBalancerEndpointsSync is true, we use the endpoint port instead
		// of the service port because we're registering each endpoint as a separate service instance.
		protocol := baseService.MicroService.Protocol()
		port := baseService.MicroService.EndpointPort()
		t.choosePorts(subset, overridePortName, overridePortNumber, protocol, port)
		if protocol.Empty() || *port == 0 {
			log.Error().Msgf("invalid port:%d or invalid protocol:%s", *port, *protocol)
			continue
		}
		for _, subsetAddr := range subset.Addresses {
			addr := new(connector.MicroServiceAddress)
			t.chooseServiceAddrPort(key, addr, port, subsetAddr, useHostname)
			if len(*addr) == 0 || !t.filterIPRanges(string(*addr)) || t.excludeIPRanges(string(*addr)) {
				continue
			}

			viaAddr := new(connector.MicroServiceAddress)
			viaPort := new(connector.MicroServicePort)
			if t.controller.GetK2CWithGateway() {
				viaAddr.Set(t.controller.GetViaIngressAddr())
				switch *protocol {
				case connector.ProtocolHTTP:
					viaPort.Set(int32(t.controller.GetViaIngressHTTPPort()))
				case connector.ProtocolGRPC:
					viaPort.Set(int32(t.controller.GetViaIngressGRPCPort()))
				default:
				}
				if t.controller.GetK2CWithGatewayMode() == ctv1.Proxy {
					addr.Set(t.controller.GetViaIngressAddr())
				}
			}

			// Its not clear whether K8S guarantees ready addresses to
			// be unique so we maintain a set to prevent duplicates just
			// in case.
			if _, has := seen[*addr]; has {
				continue
			}
			seen[*addr] = struct{}{}

			r := baseNode
			r.Service = t.bindService(svcMeta, baseService, baseService.MicroService.Service, protocol, addr, port, viaAddr, viaPort)
			// Deepcopy baseService.Meta into r.RegisteredInstances.Meta as baseService is shared
			// between all nodes of a service
			for k, v := range baseService.Meta {
				r.Service.Meta[k] = v
			}
			if subsetAddr.TargetRef != nil {
				r.Service.Meta[connector.CloudK8SRefValue] = subsetAddr.TargetRef.Name
				r.Service.Meta[connector.CloudK8SRefKind] = subsetAddr.TargetRef.Kind
			}
			if subsetAddr.NodeName != nil {
				r.Service.Meta[connector.CloudK8SNodeName] = *subsetAddr.NodeName
			}

			r.Check = &connector.AgentCheck{
				CheckID:   healthCheckID(endpoints.Namespace, t.controller.GetServiceInstanceID(r.Service.MicroService.Service, string(*addr), *port, *protocol)),
				Name:      cloudKubernetesCheckName,
				Namespace: baseService.MicroService.Namespace,
				Type:      cloudKubernetesCheckType,
				Status:    connector.HealthPassing,
				ServiceID: t.controller.GetServiceInstanceID(r.Service.MicroService.Service, string(*addr), *port, *protocol),
				Output:    kubernetesSuccessReasonMsg,
			}

			t.controller.GetK2CContext().RegisteredServiceMap.Upsert(
				key,
				[]*connector.CatalogRegistration{&r},
				t.joinCatalogRegistrations)
		}
	}
}

func (t *KtoCSource) choosePorts(subset corev1.EndpointSubset,
	overridePortName string,
	overridePortNumber int32,
	protocol *connector.MicroServiceProtocol,
	port *connector.MicroServicePort) {
	if overridePortName != "" {
		// If we're supposed to use a specific named port, find it.
		for _, p := range subset.Ports {
			if overridePortName == p.Name {
				port.Set(p.Port)
				if protocol.Empty() && p.AppProtocol != nil {
					protocol.Set(strings.ToLower(*p.AppProtocol))
				}
				break
			}
		}
	} else if overridePortNumber == 0 {
		// Otherwise we'll just use the first port in the list
		// (unless the port number was overridden by an annotation).
		for _, p := range subset.Ports {
			if *port == 0 && p.AppProtocol != nil &&
				strings.EqualFold(strings.ToUpper(string(p.Protocol)), strings.ToUpper(constants.ProtocolTCP)) {
				if protocol.Empty() {
					protocol.Set(strings.ToLower(*p.AppProtocol))
					port.Set(p.Port)
				} else if strings.EqualFold(strings.ToLower(*p.AppProtocol), strings.ToLower(protocol.Get())) {
					port.Set(p.Port)
				}
			}
			if *port > 0 {
				break
			}
		}
	}
}

func (t *KtoCSource) excludeIPRanges(addr string) (exclude bool) {
	if ipRanges := t.controller.GetK2CExcludeIPRanges(); len(ipRanges) > 0 {
		for _, cidr := range ipRanges {
			if cidr.Contains(addr) {
				exclude = true
				break
			}
		}
	}
	return
}

func (t *KtoCSource) filterIPRanges(addr string) (include bool) {
	if ipRanges := t.controller.GetK2CFilterIPRanges(); len(ipRanges) > 0 {
		for _, cidr := range ipRanges {
			if cidr.Contains(addr) {
				include = true
				break
			}
		}
	} else {
		include = true
	}
	return
}

func (t *KtoCSource) chooseServiceAddrPort(key string,
	addr *connector.MicroServiceAddress,
	port *connector.MicroServicePort,
	subsetAddr corev1.EndpointAddress,
	useHostname bool) {
	// Use the address and port from the Ingress resource if
	// ingress-sync is enabled and the service has an ingress
	// resource that references it.
	if t.controller.GetSyncIngress() && t.isIngressService(key) {
		if svcAddr, exists := t.controller.GetK2CContext().ServiceHostnameMap.Get(key); exists {
			addr.Set(svcAddr.HostName)
			port.Set(svcAddr.Port)
		}
	} else {
		addr.Set(subsetAddr.IP)
		if len(*addr) == 0 && useHostname {
			addr.Set(subsetAddr.Hostname)
		}
	}
}

func (t *KtoCSource) bindService(
	svcMeta *connector.MicroSvcMeta,
	baseService connector.AgentService,
	service string,
	protocol *connector.MicroServiceProtocol,
	addr *connector.MicroServiceAddress,
	port *connector.MicroServicePort,
	viaAddr *connector.MicroServiceAddress,
	viaPort *connector.MicroServicePort) *connector.AgentService {
	rs := baseService
	rs.ID = t.controller.GetServiceInstanceID(service, string(*addr), *port, *protocol)
	rs.MicroService.Protocol().SetVar(*protocol)
	rs.MicroService.Endpoint().Set(*addr, *port)
	rs.MicroService.Via().Set(*viaAddr, *viaPort)
	rs.Meta = make(map[string]interface{})
	rs.Meta[connector.CloudViaGatewayMode] = string(t.controller.GetK2CWithGatewayMode())
	if *protocol == connector.ProtocolHTTP {
		rs.Meta[connector.CloudHTTPViaGateway] = fmt.Sprintf("%s:%d", *viaAddr, *viaPort)
	}
	if *protocol == connector.ProtocolGRPC {
		rs.Meta[connector.CloudGRPCViaGateway] = fmt.Sprintf("%s:%d", *viaAddr, *viaPort)
		if svcMeta != nil && svcMeta.GRPCMeta != nil {
			if len(svcMeta.GRPCMeta.Interface) > 0 {
				rs.GRPCInterface = svcMeta.GRPCMeta.Interface
				if len(svcMeta.GRPCMeta.Methods) > 0 {
					for method := range svcMeta.GRPCMeta.Methods {
						rs.GRPCMethods = append(rs.GRPCMethods, method)
					}
				}
				if svcMeta.Endpoints != nil {
					if endpointMeta, exists := svcMeta.Endpoints[*addr]; exists {
						rs.GRPCInstanceMeta = endpointMeta.GRPCMeta
					}
				}
			}
		}
	}
	return &rs
}

func (t *KtoCSource) joinCatalogRegistrations(exist bool,
	valueInMap, newValue []*connector.CatalogRegistration) []*connector.CatalogRegistration {
	return append(valueInMap, newValue...)
}

// sync calls the Syncer.Sync function from the generated registrations.
//
// Precondition: lock must be held.
func (t *KtoCSource) sync() {
	atomic.AddUint64(&t.serviceDetas, 1)
	t.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
		Kind:   announcements.ServiceUpdate,
		NewObj: nil,
		OldObj: nil,
	})
}

// serviceEndpointsSource implements controller.Resource and starts
// a background watcher on endpoints that is used by the KtoCSource
// to keep track of changing endpoints for registered services.
type serviceEndpointsSource struct {
	Service  *KtoCSource
	Ctx      context.Context
	Resource connector.Resource
	informer cache.SharedIndexInformer
}

// Run implements the controller.Backgrounder interface.
func (t *serviceEndpointsSource) Run(ch <-chan struct{}) {
	log.Info().Msg("starting runner for ingress")
	(&connector.CacheController{
		Resource: t.Resource,
	}).Run(ch)
}

func (t *serviceEndpointsSource) Informer() cache.SharedIndexInformer {
	// Watch all k8s namespaces. Events will be filtered out as appropriate in the
	// `shouldTrackEndpoints` function which checks whether the service is marked
	// to be tracked by the `shouldSync` function which uses the allow and deny
	// namespace lists.
	if t.informer == nil {
		t.informer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return t.Service.kubeClient.CoreV1().
						Endpoints(metav1.NamespaceAll).
						List(t.Ctx, options)
				},

				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return t.Service.kubeClient.CoreV1().
						Endpoints(metav1.NamespaceAll).
						Watch(t.Ctx, options)
				},
			},
			&corev1.Endpoints{},
			0,
			cache.Indexers{},
		)
	}
	return t.informer
}

func (t *serviceEndpointsSource) getEndpoints(key string) (*corev1.Endpoints, error) {
	item, exists, err := t.informer.GetIndexer().GetByKey(key)
	if err == nil && exists {
		return item.(*corev1.Endpoints), nil
	}
	return nil, err
}

func (t *serviceEndpointsSource) Upsert(key string, raw interface{}) error {
	svc := t.Service
	endpoints, ok := raw.(*corev1.Endpoints)
	if !ok {
		log.Warn().Msgf("upsert got invalid type raw:%v", raw)
		return nil
	}

	if len(endpoints.Subsets) == 0 {
		return nil
	}

	svc.Lock()
	defer svc.Unlock()

	// Check if we care about endpoints for this service
	if !svc.shouldTrackEndpoints(key) {
		return nil
	}

	svc.controller.GetK2CContext().EndpointsMap.Set(key, endpoints)

	// Update the registration and trigger a sync
	svc.generateRegistrations(key)
	svc.sync()
	log.Info().Msgf("upsert endpoint key:%s", key)
	return nil
}

func (t *serviceEndpointsSource) Delete(key string, _ interface{}) error {
	t.Service.Lock()
	defer t.Service.Unlock()

	// This is a bit of an optimization. We only want to force a resync
	// if we were tracking this endpoint to begin with and that endpoint
	// had associated registrations.
	t.Service.controller.GetK2CContext().EndpointsMap.Remove(key)
	t.Service.controller.GetK2CContext().RegisteredServiceMap.Remove(key)
	t.Service.sync()

	log.Info().Msgf("delete endpoint key:%s", key)
	return nil
}

// serviceIngressSource implements controller.Resource and starts
// a background watcher on ingress resources that is used by the KtoCSource
// to keep track of changing ingress for registered services.
type serviceIngressSource struct {
	Service  *KtoCSource
	Resource connector.Resource
	Ctx      context.Context
}

func (t *serviceIngressSource) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return t.Service.kubeClient.NetworkingV1().
					Ingresses(metav1.NamespaceAll).
					List(t.Ctx, options)
			},

			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return t.Service.kubeClient.NetworkingV1().
					Ingresses(metav1.NamespaceAll).
					Watch(t.Ctx, options)
			},
		},
		&networkingv1.Ingress{},
		0,
		cache.Indexers{},
	)
}

func (t *serviceIngressSource) Upsert(key string, raw interface{}) error {
	if !t.Service.controller.GetSyncIngress() {
		return nil
	}
	svc := t.Service
	ingress, ok := raw.(*networkingv1.Ingress)
	if !ok {
		log.Warn().Msgf("upsert got invalid type raw:%v", raw)
		return nil
	}

	svc.Lock()
	defer svc.Unlock()

	for _, rule := range ingress.Spec.Rules {
		var svcName string
		var hostName string
		var svcPort int32
		for _, path := range rule.HTTP.Paths {
			if path.Path == "/" {
				svcName = path.Backend.Service.Name
				svcPort = 80
			} else {
				continue
			}
		}
		if svcName == "" {
			continue
		}
		if t.Service.controller.GetSyncIngressLoadBalancerIPs() {
			if len(ingress.Status.LoadBalancer.Ingress) > 0 && ingress.Status.LoadBalancer.Ingress[0].IP == "" {
				continue
			}
			hostName = ingress.Status.LoadBalancer.Ingress[0].IP
		} else {
			hostName = rule.Host
		}
		for _, ingressTLS := range ingress.Spec.TLS {
			for _, host := range ingressTLS.Hosts {
				if rule.Host == host {
					svcPort = 443
				}
			}
		}

		// Maintain a list of the service name to the hostname from the Ingress resource.
		svc.controller.GetK2CContext().ServiceHostnameMap.Set(
			fmt.Sprintf("%s/%s", ingress.Namespace, svcName),
			connector.ServiceAddress{
				HostName: hostName,
				Port:     svcPort,
			})
		set, exists := svc.controller.GetK2CContext().IngressServiceMap.Get(key)
		if !exists {
			svc.controller.GetK2CContext().IngressServiceMap.SetIfAbsent(key, connector.NewConcurrentMap[string]())
			set, _ = svc.controller.GetK2CContext().IngressServiceMap.Get(key)
		}
		// Maintain a list of all the service names that map to an Ingress resource.
		set.SetIfAbsent(fmt.Sprintf("%s/%s", ingress.Namespace, svcName), "")
	}

	// Update the registration for each matched service and trigger a sync
	if set, exists := svc.controller.GetK2CContext().IngressServiceMap.Get(key); exists {
		for item := range set.IterBuffered() {
			svcName := item.Key
			log.Info().Msgf("generating registrations for %s", svcName)
			svc.generateRegistrations(svcName)
		}
		svc.sync()
	}

	log.Info().Msgf("upsert ingress key:%s", key)

	return nil
}

func (t *serviceIngressSource) Delete(key string, _ interface{}) error {
	if !t.Service.controller.GetSyncIngress() {
		return nil
	}

	t.Service.Lock()
	defer t.Service.Unlock()

	// This is a bit of an optimization. We only want to force a resync
	// if we were tracking this ingress to begin with and that ingress
	// had associated registrations.
	if set, ok := t.Service.controller.GetK2CContext().IngressServiceMap.Get(key); ok {
		for item := range set.IterBuffered() {
			svcName := item.Key
			t.Service.controller.GetK2CContext().ServiceHostnameMap.Remove(svcName)
		}
		t.Service.controller.GetK2CContext().IngressServiceMap.Remove(key)
		t.Service.sync()
	}

	log.Info().Msgf("delete ingress key:%s", key)
	return nil
}

func (t *KtoCSource) addPrefixAndK8SNamespace(name, namespace string) string {
	if addServicePrefix := t.controller.GetAddServicePrefix(); len(addServicePrefix) > 0 {
		name = fmt.Sprintf("%s%s", addServicePrefix, name)
	}

	if t.controller.GetAddK8SNamespaceAsServiceSuffix() {
		name = fmt.Sprintf("%s-%s", name, namespace)
	}

	return name
}

// isIngressService return if a service has an Ingress resource that references it.
func (t *KtoCSource) isIngressService(key string) bool {
	svcAddr, ok := t.controller.GetK2CContext().ServiceHostnameMap.Get(key)
	if ok && len(svcAddr.HostName) > 0 {
		return true
	}
	return false
}

// healthCheckID deterministically generates a health check ID based on service ID and Kubernetes namespace.
func healthCheckID(k8sNS string, serviceID string) string {
	return fmt.Sprintf("%s/%s", k8sNS, serviceID)
}

// Calculates the passing service weight.
func getServiceWeight(weight string) (int, error) {
	// error validation if the input param is a number.
	weightI, err := strconv.Atoi(weight)
	if err != nil {
		return -1, err
	}

	if weightI <= 1 {
		return -1, fmt.Errorf("expecting the service annotation %s value to be greater than 1", connector.AnnotationServiceWeight)
	}

	return weightI, nil
}

// parseTags parses the tags annotation into a slice of tags.
// Tags are split on commas (except for escaped commas "\,").
func parseTags(tagsAnno string) []string {
	// This algorithm parses the tagsAnno string into a slice of strings.
	// Ideally we'd just split on commas but since Consul tags support commas,
	// we allow users to escape commas so they're included in the tag, e.g.
	// the annotation "tag\,with\,commas,tag2" will become the tags:
	// ["tag,with,commas", "tag2"].

	var tags []string
	// nextTag is built up char by char until we see a comma. Then we
	// append it to tags.
	var nextTag string

	for _, runeChar := range tagsAnno {
		runeStr := fmt.Sprintf("%c", runeChar)

		// Not a comma, just append to nextTag.
		if runeStr != "," {
			nextTag += runeStr
			continue
		}

		// Reached a comma but there's nothing in nextTag,
		// skip. (e.g. "a,,b" => ["a", "b"])
		if len(nextTag) == 0 {
			continue
		}

		// Check if the comma was escaped comma, e.g. "a\,b".
		if string(nextTag[len(nextTag)-1]) == `\` {
			// Replace the backslash with a comma.
			nextTag = nextTag[0:len(nextTag)-1] + ","
			continue
		}

		// Non-escaped comma. We're ready to push nextTag onto tags and reset nextTag.
		tags = append(tags, strings.TrimSpace(nextTag))
		nextTag = ""
	}

	// We're done the loop but nextTag still contains the last tag.
	if len(nextTag) > 0 {
		tags = append(tags, strings.TrimSpace(nextTag))
	}

	return tags
}
