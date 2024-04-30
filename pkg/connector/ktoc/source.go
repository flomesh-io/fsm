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
	service, ok := raw.(*corev1.Service)
	if !ok {
		log.Warn().Msgf("upsert got invalid type raw:%v", raw)
		return nil
	}

	t.Lock()
	defer t.Unlock()

	if !t.shouldSync(service) {
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
	t.controller.GetK2CContext().ServiceMap.Set(key, service)
	log.Debug().Msgf("[KtoCSource.Upsert] adding service to serviceMap key:%s service:%v", key, service)

	// If we care about endpoints, we should do the initial endpoints load.
	if t.shouldTrackEndpoints(key) {
		endpoints, err := t.endpointsResource.getEndpoints(key)
		if err != nil {
			log.Debug().Msgf("error loading initial endpoints key%s err:%v",
				key,
				err)
		} else {
			t.controller.GetK2CContext().EndpointsMap.Set(key, endpoints)
			log.Debug().Msgf("[KtoCSource.Upsert] adding service's endpoints to endpointsMap key:%s service:%v endpoints:%v", key, service, endpoints)
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

	if clusterSet, ok := svc.Annotations[connector.AnnotationCloudServiceClusterSet]; ok {
		if len(clusterSet) > 0 && !strings.EqualFold(clusterSet, t.controller.GetClusterSet()) {
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
			Service: t.addPrefixAndK8SNamespace(svc.Name, svc.Namespace),
		},
		Meta: map[string]interface{}{
			connector.ClusterSetKey: t.controller.GetClusterSet(),
			connector.ConnectUIDKey: t.controller.GetConnectorUID(),
			connector.CloudK8SNS:    svc.Namespace,
		},
	}

	// If the name is explicitly annotated, adopt that name
	if v, ok := svc.Annotations[connector.AnnotationServiceName]; ok {
		baseService.Service = strings.TrimSpace(v)
	} else if v, ok := svc.Annotations[connector.AnnotationCloudServiceInheritedFrom]; ok {
		baseService.Service = strings.TrimSpace(v)
	}

	// Update the service namespace based on namespace settings
	registeredNS := t.discClient.RegisteredNamespace(svc.Namespace)
	if registeredNS != "" {
		log.Debug().Msgf("[generateRegistrations] namespace being used key:%s namespace:%s", key, registeredNS)
		baseService.Namespace = registeredNS
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
			baseService.Service,
			baseService.Namespace,
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
		t.generateLoadBalanceEndpointsRegistrations(key, baseNode, baseService, overridePortName, overridePortNumber, svc)

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
		t.registerServiceInstance(baseNode, baseService, key, overridePortName, overridePortNumber, true)
	}
}

func (t *KtoCSource) determinePortAnnotations(svc *corev1.Service, baseService connector.AgentService) (string, int) {
	var overridePortName string
	var overridePortNumber int
	if len(svc.Spec.Ports) > 0 {
		var port int
		isNodePort := svc.Spec.Type == corev1.ServiceTypeNodePort

		// If a specific port is specified, then use that port value
		portAnnotation, ok := svc.Annotations[connector.AnnotationServicePort]
		if ok {
			if v, err := strconv.ParseInt(portAnnotation, 0, 0); err == nil {
				port = int(v)
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
						port = int(p.NodePort)
					} else {
						port = int(p.Port)
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
						port = int(p.NodePort)
						break
					}
				}
			} else {
				port = int(svc.Spec.Ports[0].Port)
				// NOTE: for cluster IP services we always use the endpoint
				// ports so this will be overridden.
			}
		}

		baseService.HTTPPort = port

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
			r.Service.ID = t.controller.GetServiceInstanceID(r.Service.Service, ip, rs.HTTPPort, rs.GRPCPort)
			r.Service.Address = ip
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
					found = true
					r := baseNode
					rs := baseService
					r.Service = &rs
					r.Service.ID = t.controller.GetServiceInstanceID(r.Service.Service, subsetAddr.IP, rs.HTTPPort, rs.GRPCPort)
					r.Service.Address = address.Address

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
						r := baseNode
						rs := baseService
						r.Service = &rs
						r.Service.ID = t.controller.GetServiceInstanceID(r.Service.Service, subsetAddr.IP, rs.HTTPPort, rs.GRPCPort)
						r.Service.Address = address.Address

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

func (t *KtoCSource) generateLoadBalanceEndpointsRegistrations(key string, baseNode connector.CatalogRegistration, baseService connector.AgentService, overridePortName string, overridePortNumber int, svc *corev1.Service) {
	if t.controller.GetSyncLoadBalancerEndpoints() {
		t.registerServiceInstance(baseNode, baseService, key, overridePortName, overridePortNumber, false)
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

			if _, ok := seen[addr]; ok {
				continue
			}
			seen[addr] = struct{}{}

			r := baseNode
			rs := baseService
			r.Service = &rs
			r.Service.ID = t.controller.GetServiceInstanceID(r.Service.Service, addr, rs.HTTPPort, rs.GRPCPort)
			r.Service.Address = addr

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
	baseNode connector.CatalogRegistration,
	baseService connector.AgentService,
	key string,
	overridePortName string,
	overridePortNumber int,
	useHostname bool) {
	if t.controller.GetK2CContext().EndpointsMap.IsEmpty() {
		return
	}

	endpoints, ok := t.controller.GetK2CContext().EndpointsMap.Get(key)
	if !ok || endpoints == nil || len(endpoints.Subsets) == 0 {
		return
	}

	seen := map[string]struct{}{}
	for _, subset := range endpoints.Subsets {
		// For ClusterIP services and if loadBalancerEndpointsSync is true, we use the endpoint port instead
		// of the service port because we're registering each endpoint
		// as a separate service instance.
		httpPort := baseService.HTTPPort
		grpcPort := baseService.GRPCPort
		if overridePortName != "" {
			// If we're supposed to use a specific named port, find it.
			for _, p := range subset.Ports {
				if overridePortName == p.Name {
					httpPort = int(p.Port)
					break
				}
			}
		} else if overridePortNumber == 0 {
			// Otherwise we'll just use the first port in the list
			// (unless the port number was overridden by an annotation).
			for _, p := range subset.Ports {
				if httpPort == 0 &&
					strings.EqualFold(string(p.Protocol), strings.ToUpper(constants.ProtocolTCP)) &&
					p.AppProtocol != nil &&
					strings.EqualFold(*p.AppProtocol, strings.ToUpper(constants.ProtocolHTTP)) {
					httpPort = int(p.Port)
				}
				if grpcPort == 0 &&
					strings.EqualFold(string(p.Protocol), strings.ToUpper(constants.ProtocolTCP)) &&
					p.AppProtocol != nil &&
					strings.EqualFold(*p.AppProtocol, strings.ToUpper(constants.ProtocolGRPC)) {
					grpcPort = int(p.Port)
				}
				if httpPort > 0 && grpcPort > 0 {
					break
				}
			}
		}
		for _, subsetAddr := range subset.Addresses {
			var addr string
			var viaAddr string
			var viaPort int
			addr, httpPort = t.chooseServiceAddrPort(key, httpPort, subsetAddr, useHostname)
			if len(addr) == 0 {
				continue
			}

			if t.controller.GetK2CWithGateway() {
				if t.controller.GetK2CWithGatewayMode() == ctv1.Forward {
					viaAddr = t.controller.GetViaIngressAddr()
					viaPort = int(t.controller.GetViaIngressHTTPPort())
				}
				if t.controller.GetK2CWithGatewayMode() == ctv1.Proxy {
					addr = t.controller.GetViaIngressAddr()
					httpPort = int(t.controller.GetViaIngressHTTPPort())
				}
			}

			// Its not clear whether K8S guarantees ready addresses to
			// be unique so we maintain a set to prevent duplicates just
			// in case.
			if _, has := seen[addr]; has {
				continue
			}
			seen[addr] = struct{}{}

			r := baseNode
			r.Service = t.bindService(baseService, baseService.Service, addr, httpPort, grpcPort, viaAddr, viaPort)
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
				CheckID:   healthCheckID(endpoints.Namespace, t.controller.GetServiceInstanceID(r.Service.Service, addr, httpPort, grpcPort)),
				Name:      cloudKubernetesCheckName,
				Namespace: baseService.Namespace,
				Type:      cloudKubernetesCheckType,
				Status:    connector.HealthPassing,
				ServiceID: t.controller.GetServiceInstanceID(r.Service.Service, addr, httpPort, grpcPort),
				Output:    kubernetesSuccessReasonMsg,
			}

			t.controller.GetK2CContext().RegisteredServiceMap.Upsert(
				key,
				[]*connector.CatalogRegistration{&r},
				t.joinCatalogRegistrations)
		}
	}
}

func (t *KtoCSource) chooseServiceAddrPort(key string, port int, subsetAddr corev1.EndpointAddress, useHostname bool) (addr string, httpPort int) {
	// Use the address and port from the Ingress resource if
	// ingress-sync is enabled and the service has an ingress
	// resource that references it.
	if t.controller.GetSyncIngress() && t.isIngressService(key) {
		if svcAddr, exists := t.controller.GetK2CContext().ServiceHostnameMap.Get(key); exists {
			addr = svcAddr.HostName
			httpPort = int(svcAddr.Port)
		}
	} else {
		addr = subsetAddr.IP
		if len(addr) == 0 && useHostname {
			addr = subsetAddr.Hostname
		}
		httpPort = port
	}
	return addr, httpPort
}

func (t *KtoCSource) bindService(baseService connector.AgentService,
	service, addr string, httpPort, grpcPort int,
	viaAddr string, viaPort int) *connector.AgentService {
	rs := baseService
	rs.ID = t.controller.GetServiceInstanceID(service, addr, httpPort, grpcPort)
	rs.Address = addr
	rs.HTTPPort = httpPort
	rs.GRPCPort = grpcPort
	rs.ViaAddress = viaAddr
	rs.ViaPort = viaPort
	rs.Meta = make(map[string]interface{})
	if len(viaAddr) > 0 && viaPort > 0 {
		rs.Meta[connector.CloudK8SVia] = fmt.Sprintf("%s:%d", viaAddr, viaPort)
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
