package ktoc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
	"github.com/flomesh-io/fsm/pkg/constants"
)

const (
	// CloudK8SNS is the key used in the meta to record the namespace
	// of the service/node registration.
	CloudK8SNS       = "fsm-connector-external-k8s-ns"
	CloudK8SRefKind  = "fsm-connector-external-k8s-ref-kind"
	CloudK8SRefValue = "fsm-connector-external-k8s-ref-name"
	CloudK8SNodeName = "fsm-connector-external-k8s-node-name"

	// cloudKubernetesCheckType is the type of health check in cloud for Kubernetes readiness status.
	cloudKubernetesCheckType = "kubernetes-readiness"
	// cloudKubernetesCheckName is the name of health check in cloud for Kubernetes readiness status.
	cloudKubernetesCheckName   = "Kubernetes Readiness Check"
	kubernetesSuccessReasonMsg = "Kubernetes health checks passing"
)

type NodePortSyncType string

const (
	// ExternalOnly only sync NodePort services with a node's ExternalIP address.
	// Doesn't sync if an ExternalIP doesn't exist.
	ExternalOnly NodePortSyncType = "ExternalOnly"

	// ExternalFirst sync with an ExternalIP first, if it doesn't exist, use the
	// node's InternalIP address instead.
	ExternalFirst NodePortSyncType = "ExternalFirst"

	// InternalOnly sync NodePort services using.
	InternalOnly NodePortSyncType = "InternalOnly"
)

// ServiceResource implements controller.Resource to sync CatalogService resource
// types from K8S.
type ServiceResource struct {
	Client kubernetes.Interface
	Syncer Syncer

	// Ctx is used to cancel processes kicked off by ServiceResource.
	Ctx context.Context

	// AllowK8sNamespacesSet is a set of k8s namespaces to explicitly allow for
	// syncing. It supports the special character `*` which indicates that
	// all k8s namespaces are eligible unless explicitly denied. This filter
	// is applied before checking pod annotations.
	AllowK8sNamespacesSet mapset.Set

	// DenyK8sNamespacesSet is a set of k8s namespaces to explicitly deny
	// syncing and thus service registration with Consul. An empty set
	// means that no namespaces are removed from consideration. This filter
	// takes precedence over AllowK8sNamespacesSet.
	DenyK8sNamespacesSet mapset.Set

	// ConsulK8STag is the tag value for services registered.
	ConsulK8STag string

	//AddServicePrefix prepends K8s services in cloud with a prefix
	AddServicePrefix string

	// ExplictEnable should be set to true to require explicit enabling
	// using annotations. If this is false, then services are implicitly
	// enabled (aka default enabled).
	ExplicitEnable bool

	// ClusterIPSync set to true (the default) syncs ClusterIP-type services.
	// Setting this to false will ignore ClusterIP services during the sync.
	ClusterIPSync bool

	// LoadBalancerEndpointsSync set to true (default false) will sync ServiceTypeLoadBalancer endpoints.
	LoadBalancerEndpointsSync bool

	// NodeExternalIPSync set to true (the default) syncs NodePort services
	// using the node's external ip address. When false, the node's internal
	// ip address will be used instead.
	NodePortSync NodePortSyncType

	// AddK8SNamespaceAsServiceSuffix set to true appends Kubernetes namespace
	// to the service name being synced to Cloud separated by a dash.
	// For example, service 'foo' in the 'default' namespace will be synced
	// as 'foo-default'.
	AddK8SNamespaceAsServiceSuffix bool

	// EnableNamespaces indicates that a user is running Consul Enterprise
	// with version 1.7+ which is namespace aware. It enables Consul namespaces,
	// with syncing into either a single Consul namespace or mirrored from
	// k8s namespaces.
	EnableNamespaces bool

	// ConsulDestinationNamespace is the name of the Consul namespace to register all
	// synced services into if Consul namespaces are enabled and mirroring
	// is disabled. This will not be used if mirroring is enabled.
	ConsulDestinationNamespace string

	// EnableK8SNSMirroring causes Consul namespaces to be created to match the
	// organization within k8s. Services are registered into the Consul
	// namespace that mirrors their k8s namespace.
	EnableK8SNSMirroring bool

	// K8SNSMirroringPrefix is an optional prefix that can be added to the Consul
	// namespaces created while mirroring. For example, if it is set to "k8s-",
	// then the k8s `default` namespace will be mirrored in Consul's
	// `k8s-default` namespace.
	K8SNSMirroringPrefix string

	// The Consul node name to register service with.
	ConsulNodeName string

	// serviceLock must be held for any read/write to these maps.
	serviceLock sync.RWMutex

	// serviceMap holds services we should sync to cloud. Keys are the
	// in the form <kube namespace>/<kube svc name>.
	serviceMap map[string]*corev1.Service

	// endpointsMap uses the same keys as serviceMap but maps to the endpoints
	// of each service.
	endpointsMap map[string]*corev1.Endpoints

	// SyncIngress enables syncing of the hostname from an Ingress resource
	// to the service registration if an Ingress rule matches the service.
	SyncIngress bool

	// SyncIngressLoadBalancerIPs enables syncing the IP of the Ingress LoadBalancer
	// if we do not want to sync the hostname from the Ingress resource.
	SyncIngressLoadBalancerIPs bool

	// ingressServiceMap uses the same keys as serviceMap but maps to the ingress
	// of each service if it exists.
	ingressServiceMap map[string]map[string]string

	// serviceHostnameMap maps the name of a service to the hostName and port that
	// is provided by the Ingress resource for the service.
	serviceHostnameMap map[string]serviceAddress

	// registeredServiceMap holds the services in cloud that we've registered from kube.
	// It's populated via cloud's API and lets us diff what is actually in
	// cloud vs. what we expect to be there.
	registeredServiceMap map[string][]*provider.CatalogRegistration
}

type serviceAddress struct {
	hostName string
	port     int32
}

// Informer implements the controller.Resource interface.
func (t *ServiceResource) Informer() cache.SharedIndexInformer {
	// Watch all k8s namespaces. Events will be filtered out as appropriate
	// based on the allow and deny lists in the `shouldSync` function.
	return cache.NewSharedIndexInformer(
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

// Upsert implements the controller.Resource interface.
func (t *ServiceResource) Upsert(key string, raw interface{}) error {
	// We expect a CatalogService. If it isn't a service then just ignore it.
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
		if _, ok := t.serviceMap[key]; ok {
			log.Info().Msgf("service should no longer be synced service:%s", key)
			t.doDelete(key)
		} else {
			log.Debug().Msgf("[ServiceResource.Upsert] syncing disabled for service, ignoring key:%s", key)
		}
		return nil
	}

	// Syncing is enabled, let's keep track of this service.
	t.serviceMap[key] = service
	log.Debug().Msgf("[ServiceResource.Upsert] adding service to serviceMap key:%s service:%v", key, service)

	// If we care about endpoints, we should do the initial endpoints load.
	if t.shouldTrackEndpoints(key) {
		endpoints, err := t.Client.CoreV1().
			Endpoints(service.Namespace).
			Get(t.Ctx, service.Name, metav1.GetOptions{})
		if err != nil {
			log.Warn().Msgf("error loading initial endpoints key%s err:%v",
				key,
				err)
		} else {
			if t.endpointsMap == nil {
				t.endpointsMap = make(map[string]*corev1.Endpoints)
			}
			t.endpointsMap[key] = endpoints
			log.Debug().Msgf("[ServiceResource.Upsert] adding service's endpoints to endpointsMap key:%s service:%v endpoints:%v", key, service, endpoints)
		}
	}

	// Update the registration and trigger a sync
	t.generateRegistrations(key)
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
	log.Debug().Msgf("[doDelete] deleting service from serviceMap key:%s", key)
	delete(t.endpointsMap, key)
	log.Debug().Msgf("[doDelete] deleting endpoints from endpointsMap key:%s", key)
	// If there were registrations related to this service, then
	// delete them and sync.
	if _, ok := t.registeredServiceMap[key]; ok {
		delete(t.registeredServiceMap, key)
		t.sync()
	}
}

// Run implements the controller.Backgrounder interface.
func (t *ServiceResource) Run(ch <-chan struct{}) {
	log.Info().Msg("starting runner for endpoints")
	// Register a controller for Endpoints which subsequently registers a
	// controller for the Ingress resource.
	(&Controller{
		Resource: &serviceEndpointsResource{
			Service: t,
			Ctx:     t.Ctx,
			Resource: &serviceIngressResource{
				Service:                    t,
				Ctx:                        t.Ctx,
				SyncIngressLoadBalancerIPs: t.SyncIngressLoadBalancerIPs,
				EnableIngress:              t.SyncIngress,
			},
		},
	}).Run(ch)
}

// shouldSync returns true if resyncing should be enabled for the given service.
func (t *ServiceResource) shouldSync(svc *corev1.Service) bool {
	// Namespace logic
	if svc.Namespace == syncCloudNamespace {
		log.Debug().Msgf("[shouldSync] service is in the deny list svc.Namespace:%s service:%v", svc.Namespace, svc)
		return false
	}

	// If in deny list, don't sync
	if t.DenyK8sNamespacesSet.Contains(svc.Namespace) {
		log.Debug().Msgf("[shouldSync] service is in the deny list svc.Namespace:%s service:%v", svc.Namespace, svc)
		return false
	}

	// If not in allow list or allow list is not *, don't sync
	if !t.AllowK8sNamespacesSet.Contains("*") && !t.AllowK8sNamespacesSet.Contains(svc.Namespace) {
		log.Debug().Msgf("[shouldSync] service not in allow list svc.Namespace:%s service:%v", svc.Namespace, svc)
		return false
	}

	// Ignore ClusterIP services if ClusterIP sync is disabled
	if svc.Spec.Type == corev1.ServiceTypeClusterIP && !t.ClusterIPSync {
		log.Debug().Msgf("[shouldSync] ignoring clusterip service svc.Namespace:%s service:%v", svc.Namespace, svc)
		return false
	}

	raw, ok := svc.Annotations[connector.AnnotationServiceSyncK8sToCloud]
	if !ok {
		// If there is no explicit value, then set it to our current default.
		return !t.ExplicitEnable
	}

	v, err := strconv.ParseBool(raw)
	if err != nil {
		log.Warn().Msgf("error parsing service-sync annotation service-name:%s err:%v",
			t.addPrefixAndK8SNamespace(svc.Name, svc.Namespace),
			err)

		// Fallback to default
		return !t.ExplicitEnable
	}

	return v
}

// shouldTrackEndpoints returns true if the endpoints for the given key
// should be tracked.
//
// Precondition: this requires the lock to be held.
func (t *ServiceResource) shouldTrackEndpoints(key string) bool {
	// The service must be one we care about for us to watch the endpoints.
	// We care about a service that exists in our service map (is enabled
	// for syncing) and is a NodePort or ClusterIP type since only those
	// types use endpoints.
	if t.serviceMap == nil {
		return false
	}
	svc, ok := t.serviceMap[key]
	if !ok {
		return false
	}

	return svc.Spec.Type == corev1.ServiceTypeNodePort ||
		svc.Spec.Type == corev1.ServiceTypeClusterIP ||
		(t.LoadBalancerEndpointsSync && svc.Spec.Type == corev1.ServiceTypeLoadBalancer)
}

// generateRegistrations generates the necessary cloud registrations for
// the given key. This is best effort: if there isn't enough information
// yet to register a service, then no registration will be generated.
//
// Precondition: the lock t.lock is held.
func (t *ServiceResource) generateRegistrations(key string) {
	// Get the service. If it doesn't exist, then we can't generate.
	svc, ok := t.serviceMap[key]
	if !ok {
		return
	}

	log.Debug().Msgf("[generateRegistrations] generating registration key:%s", key)

	// Initialize our cloud service map here if it isn't already.
	if t.registeredServiceMap == nil {
		t.registeredServiceMap = make(map[string][]*provider.CatalogRegistration)
	}

	// Begin by always clearing the old value out since we'll regenerate
	// a new one if there is one.
	delete(t.registeredServiceMap, key)

	// baseNode and baseService are the base that should be modified with
	// service-type specific changes. These are not pointers, they should be
	// shallow copied for each instance.
	baseNode := provider.CatalogRegistration{
		SkipNodeUpdate: true,
		Node:           t.ConsulNodeName,
		Address:        "127.0.0.1",
		NodeMeta: map[string]string{
			connector.ServiceSourceKey: connector.ServiceSourceValue,
		},
	}

	if withGateway {
		if len(connector.ViaGateway.IngressAddr) > 0 {
			baseNode.Address = connector.ViaGateway.IngressAddr
		}
	}

	baseService := provider.AgentService{
		Service: t.addPrefixAndK8SNamespace(svc.Name, svc.Namespace),
		Tags:    []string{t.ConsulK8STag},
		Meta: map[string]interface{}{
			connector.ServiceSourceKey: connector.ServiceSourceValue,
			CloudK8SNS:                 svc.Namespace,
		},
	}

	// If the name is explicitly annotated, adopt that name
	if v, ok := svc.Annotations[connector.AnnotationServiceName]; ok {
		baseService.Service = strings.TrimSpace(v)
	}

	// Update the Consul namespace based on namespace settings
	consulNS := provider.CloudNamespace(svc.Namespace,
		t.EnableNamespaces,
		t.ConsulDestinationNamespace,
		t.EnableK8SNSMirroring,
		t.K8SNSMirroringPrefix)
	if consulNS != "" {
		log.Debug().Msgf("[generateRegistrations] namespace being used key:%s namespace:%s", key, consulNS)
		baseService.Namespace = consulNS
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
			len(t.registeredServiceMap[key]))
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
	// If LoadBalancerEndpointsSync is true sync LB endpoints instead of loadbalancer ingress.
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

func (t *ServiceResource) determinePortAnnotations(svc *corev1.Service, baseService provider.AgentService) (string, int) {
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
			baseService.Meta["port-"+p.Name] = strconv.FormatInt(int64(p.Port), 10)
		}
	}
	return overridePortName, overridePortNumber
}

func (t *ServiceResource) generateExternalIPRegistrations(key string, svc *corev1.Service, baseNode provider.CatalogRegistration, baseService provider.AgentService) bool {
	if ips := svc.Spec.ExternalIPs; len(ips) > 0 {
		for _, ip := range ips {
			r := baseNode
			rs := baseService
			r.Service = &rs
			r.Service.ID = connector.ServiceInstanceID(r.Service.Service, ip, rs.HTTPPort)
			r.Service.Address = ip
			// Adding information about service weight.
			// Overrides the existing weight if present.
			if weight, ok := svc.Annotations[connector.AnnotationServiceWeight]; ok && weight != "" {
				weightI, err := getServiceWeight(weight)
				if err == nil {
					r.Service.Weights = provider.AgentWeights{
						Passing: weightI,
					}
				} else {
					log.Debug().Msgf("[generateRegistrations] service weight err:%v", err)
				}
			}

			t.registeredServiceMap[key] = append(t.registeredServiceMap[key], &r)
		}

		return true
	}
	return false
}

func (t *ServiceResource) generateNodeportRegistrations(key string, baseNode provider.CatalogRegistration, baseService provider.AgentService) bool {
	if t.endpointsMap == nil {
		return true
	}

	endpoints := t.endpointsMap[key]
	if endpoints == nil {
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
			node, err := t.Client.CoreV1().Nodes().Get(t.Ctx, *subsetAddr.NodeName, metav1.GetOptions{})
			if err != nil {
				log.Warn().Msgf("error getting node info error:%v", err)
				continue
			}

			// Set the expected node address type
			var expectedType corev1.NodeAddressType
			if t.NodePortSync == InternalOnly {
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
					r.Service.ID = connector.ServiceInstanceID(r.Service.Service, subsetAddr.IP, rs.HTTPPort)
					r.Service.Address = address.Address

					t.registeredServiceMap[key] = append(t.registeredServiceMap[key], &r)
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
			if t.NodePortSync == ExternalFirst && !found {
				for _, address := range node.Status.Addresses {
					if address.Type == corev1.NodeInternalIP {
						r := baseNode
						rs := baseService
						r.Service = &rs
						r.Service.ID = connector.ServiceInstanceID(r.Service.Service, subsetAddr.IP, rs.HTTPPort)
						r.Service.Address = address.Address

						t.registeredServiceMap[key] = append(t.registeredServiceMap[key], &r)
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

func (t *ServiceResource) generateLoadBalanceEndpointsRegistrations(key string, baseNode provider.CatalogRegistration, baseService provider.AgentService, overridePortName string, overridePortNumber int, svc *corev1.Service) {
	if t.LoadBalancerEndpointsSync {
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
			r.Service.ID = connector.ServiceInstanceID(r.Service.Service, addr, rs.HTTPPort)
			r.Service.Address = addr

			// Adding information about service weight.
			// Overrides the existing weight if present.
			if weight, ok := svc.Annotations[connector.AnnotationServiceWeight]; ok && weight != "" {
				weightI, err := getServiceWeight(weight)
				if err == nil {
					r.Service.Weights = provider.AgentWeights{
						Passing: weightI,
					}
				} else {
					log.Debug().Msgf("[generateRegistrations] service weight err:%v", err)
				}
			}

			t.registeredServiceMap[key] = append(t.registeredServiceMap[key], &r)
		}
	}
}

func (t *ServiceResource) registerServiceInstance(
	baseNode provider.CatalogRegistration,
	baseService provider.AgentService,
	key string,
	overridePortName string,
	overridePortNumber int,
	useHostname bool) {
	if t.endpointsMap == nil {
		return
	}

	endpoints := t.endpointsMap[key]
	if endpoints == nil {
		return
	}

	seen := map[string]struct{}{}
	for _, subset := range endpoints.Subsets {
		// For ClusterIP services and if LoadBalancerEndpointsSync is true, we use the endpoint port instead
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
					strings.EqualFold(string(*p.AppProtocol), strings.ToUpper(constants.ProtocolHTTP)) {
					httpPort = int(p.Port)
				}
				if grpcPort == 0 &&
					strings.EqualFold(string(p.Protocol), strings.ToUpper(constants.ProtocolTCP)) &&
					p.AppProtocol != nil &&
					strings.EqualFold(string(*p.AppProtocol), strings.ToUpper(constants.ProtocolGRPC)) {
					grpcPort = int(p.Port)
				}
				if httpPort > 0 && grpcPort > 0 {
					break
				}
			}
		}
		for _, subsetAddr := range subset.Addresses {
			var addr string
			// Use the address and port from the Ingress resource if
			// ingress-sync is enabled and the service has an ingress
			// resource that references it.
			if t.SyncIngress && t.isIngressService(key) {
				addr = t.serviceHostnameMap[key].hostName
				httpPort = int(t.serviceHostnameMap[key].port)
			} else {
				addr = subsetAddr.IP
				if addr == "" && useHostname {
					addr = subsetAddr.Hostname
				}
				if addr == "" {
					continue
				}
			}

			if withGateway {
				addr = connector.ViaGateway.IngressAddr
				httpPort = int(connector.ViaGateway.Ingress.HTTPPort)
			}

			// Its not clear whether K8S guarantees ready addresses to
			// be unique so we maintain a set to prevent duplicates just
			// in case.
			if _, ok := seen[addr]; ok {
				continue
			}
			seen[addr] = struct{}{}

			r := baseNode
			rs := baseService
			r.Service = &rs
			r.Service.ID = connector.ServiceInstanceID(r.Service.Service, addr, httpPort)
			r.Service.Address = addr
			r.Service.HTTPPort = httpPort
			r.Service.GRPCPort = grpcPort
			r.Service.Meta = make(map[string]interface{})
			// Deepcopy baseService.Meta into r.CatalogService.Meta as baseService is shared
			// between all nodes of a service
			for k, v := range baseService.Meta {
				r.Service.Meta[k] = v
			}
			if subsetAddr.TargetRef != nil {
				r.Service.Meta[CloudK8SRefValue] = subsetAddr.TargetRef.Name
				r.Service.Meta[CloudK8SRefKind] = subsetAddr.TargetRef.Kind
			}
			if subsetAddr.NodeName != nil {
				r.Service.Meta[CloudK8SNodeName] = *subsetAddr.NodeName
			}

			r.Check = &provider.AgentCheck{
				CheckID:   healthCheckID(endpoints.Namespace, connector.ServiceInstanceID(r.Service.Service, addr, httpPort)),
				Name:      cloudKubernetesCheckName,
				Namespace: baseService.Namespace,
				Type:      cloudKubernetesCheckType,
				Status:    provider.HealthPassing,
				ServiceID: connector.ServiceInstanceID(r.Service.Service, addr, httpPort),
				Output:    kubernetesSuccessReasonMsg,
			}

			t.registeredServiceMap[key] = append(t.registeredServiceMap[key], &r)
		}
	}
}

// sync calls the Syncer.Sync function from the generated registrations.
//
// Precondition: lock must be held.
func (t *ServiceResource) sync() {
	// NOTE(mitchellh): This isn't the most efficient way to do this and
	// the times that sync are called are also not the most efficient. All
	// of these are implementation details so lets improve this later when
	// it becomes a performance issue and just do the easy thing first.
	rs := make([]*provider.CatalogRegistration, 0, len(t.registeredServiceMap)*4)
	for _, set := range t.registeredServiceMap {
		rs = append(rs, set...)
	}

	// Sync, which should be non-blocking in real-world cases
	t.Syncer.Sync(rs)
}

// serviceEndpointsResource implements controller.Resource and starts
// a background watcher on endpoints that is used by the ServiceResource
// to keep track of changing endpoints for registered services.
type serviceEndpointsResource struct {
	Service  *ServiceResource
	Ctx      context.Context
	Resource Resource
}

// Run implements the controller.Backgrounder interface.
func (t *serviceEndpointsResource) Run(ch <-chan struct{}) {
	log.Info().Msg("starting runner for ingress")
	(&Controller{
		Resource: t.Resource,
	}).Run(ch)
}

func (t *serviceEndpointsResource) Informer() cache.SharedIndexInformer {
	// Watch all k8s namespaces. Events will be filtered out as appropriate in the
	// `shouldTrackEndpoints` function which checks whether the service is marked
	// to be tracked by the `shouldSync` function which uses the allow and deny
	// namespace lists.
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return t.Service.Client.CoreV1().
					Endpoints(metav1.NamespaceAll).
					List(t.Ctx, options)
			},

			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return t.Service.Client.CoreV1().
					Endpoints(metav1.NamespaceAll).
					Watch(t.Ctx, options)
			},
		},
		&corev1.Endpoints{},
		0,
		cache.Indexers{},
	)
}

func (t *serviceEndpointsResource) Upsert(key string, raw interface{}) error {
	svc := t.Service
	endpoints, ok := raw.(*corev1.Endpoints)
	if !ok {
		log.Warn().Msgf("upsert got invalid type raw:%v", raw)
		return nil
	}

	svc.serviceLock.Lock()
	defer svc.serviceLock.Unlock()

	// Check if we care about endpoints for this service
	if !svc.shouldTrackEndpoints(key) {
		return nil
	}

	// We are tracking this service so let's keep track of the endpoints
	if svc.endpointsMap == nil {
		svc.endpointsMap = make(map[string]*corev1.Endpoints)
	}
	svc.endpointsMap[key] = endpoints

	// Update the registration and trigger a sync
	svc.generateRegistrations(key)
	svc.sync()
	log.Info().Msgf("upsert endpoint key:%s", key)
	return nil
}

func (t *serviceEndpointsResource) Delete(key string, _ interface{}) error {
	t.Service.serviceLock.Lock()
	defer t.Service.serviceLock.Unlock()

	// This is a bit of an optimization. We only want to force a resync
	// if we were tracking this endpoint to begin with and that endpoint
	// had associated registrations.
	if _, ok := t.Service.endpointsMap[key]; ok {
		delete(t.Service.endpointsMap, key)
		if _, ok := t.Service.registeredServiceMap[key]; ok {
			delete(t.Service.registeredServiceMap, key)
			t.Service.sync()
		}
	}

	log.Info().Msgf("delete endpoint key:%s", key)
	return nil
}

// serviceIngressResource implements controller.Resource and starts
// a background watcher on ingress resources that is used by the ServiceResource
// to keep track of changing ingress for registered services.
type serviceIngressResource struct {
	Service                    *ServiceResource
	Resource                   Resource
	Ctx                        context.Context
	EnableIngress              bool
	SyncIngressLoadBalancerIPs bool
}

func (t *serviceIngressResource) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return t.Service.Client.NetworkingV1().
					Ingresses(metav1.NamespaceAll).
					List(t.Ctx, options)
			},

			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return t.Service.Client.NetworkingV1().
					Ingresses(metav1.NamespaceAll).
					Watch(t.Ctx, options)
			},
		},
		&networkingv1.Ingress{},
		0,
		cache.Indexers{},
	)
}

func (t *serviceIngressResource) Upsert(key string, raw interface{}) error {
	if !t.EnableIngress {
		return nil
	}
	svc := t.Service
	ingress, ok := raw.(*networkingv1.Ingress)
	if !ok {
		log.Warn().Msgf("upsert got invalid type raw:%v", raw)
		return nil
	}

	svc.serviceLock.Lock()
	defer svc.serviceLock.Unlock()

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
		if t.SyncIngressLoadBalancerIPs {
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

		if svc.serviceHostnameMap == nil {
			svc.serviceHostnameMap = make(map[string]serviceAddress)
		}
		// Maintain a list of the service name to the hostname from the Ingress resource.
		svc.serviceHostnameMap[fmt.Sprintf("%s/%s", ingress.Namespace, svcName)] = serviceAddress{
			hostName: hostName,
			port:     svcPort,
		}
		if svc.ingressServiceMap == nil {
			svc.ingressServiceMap = make(map[string]map[string]string)
		}
		if svc.ingressServiceMap[key] == nil {
			svc.ingressServiceMap[key] = make(map[string]string)
		}
		// Maintain a list of all the service names that map to an Ingress resource.
		svc.ingressServiceMap[key][fmt.Sprintf("%s/%s", ingress.Namespace, svcName)] = ""
	}

	// Update the registration for each matched service and trigger a sync
	for svcName := range svc.ingressServiceMap[key] {
		log.Info().Msgf("generating registrations for %s", svcName)
		svc.generateRegistrations(svcName)
	}
	svc.sync()
	log.Info().Msgf("upsert ingress key:%s", key)

	return nil
}

func (t *serviceIngressResource) Delete(key string, _ interface{}) error {
	if !t.EnableIngress {
		return nil
	}
	t.Service.serviceLock.Lock()
	defer t.Service.serviceLock.Unlock()

	// This is a bit of an optimization. We only want to force a resync
	// if we were tracking this ingress to begin with and that ingress
	// had associated registrations.
	if _, ok := t.Service.ingressServiceMap[key]; ok {
		for svcName := range t.Service.ingressServiceMap[key] {
			delete(t.Service.serviceHostnameMap, svcName)
		}
		delete(t.Service.ingressServiceMap, key)
		t.Service.sync()
	}

	log.Info().Msgf("delete ingress key:%s", key)
	return nil
}

func (t *ServiceResource) addPrefixAndK8SNamespace(name, namespace string) string {
	if t.AddServicePrefix != "" {
		name = fmt.Sprintf("%s%s", t.AddServicePrefix, name)
	}

	if t.AddK8SNamespaceAsServiceSuffix {
		name = fmt.Sprintf("%s-%s", name, namespace)
	}

	return name
}

// isIngressService return if a service has an Ingress resource that references it.
func (t *ServiceResource) isIngressService(key string) bool {
	return t.serviceHostnameMap != nil && t.serviceHostnameMap[key].hostName != ""
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
