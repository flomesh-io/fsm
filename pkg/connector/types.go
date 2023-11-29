package connector

const (
	//ConsulDiscoveryService defines consul discovery service name
	ConsulDiscoveryService = "consul"

	//EurekaDiscoveryService defines eureka discovery service name
	EurekaDiscoveryService = "eureka"
)

const (
	// ServiceSourceKey is the key used in the meta to track the "k8s" source.
	ServiceSourceKey = "fsm-connector-external-source"
)

var (
	// ServiceSourceValue is the value of the source.
	ServiceSourceValue = "kubernetes"
)

const (
	// AnnotationMeshServiceSync defines mesh service sync annotation
	AnnotationMeshServiceSync = "flomesh.io/mesh-service-sync"

	// AnnotationCloudServiceInheritedFrom defines cloud service inherited annotation
	AnnotationCloudServiceInheritedFrom = "flomesh.io/cloud-service-inherited-from"

	// AnnotationMeshEndpointAddr defines mesh endpoint addr annotation
	AnnotationMeshEndpointAddr = "flomesh.io/cloud-endpoint-addr"
)

const (
	// AnnotationServiceSync is the key of the annotation that determines
	// whether to sync the CatalogService resource or not. If this isn't set then
	// the default based on the syncer configuration is chosen.
	AnnotationServiceSync = "flomesh.io/service-sync"

	// AnnotationServiceName is set to override the name of the service
	// registered. By default this will be the name of the CatalogService resource.
	AnnotationServiceName = "flomesh.io/service-name"

	// AnnotationServicePort specifies the port to use as the service instance
	// port when registering a service. This can be a named port in the
	// service or an integer value.
	AnnotationServicePort = "flomesh.io/service-port"

	// AnnotationServiceTags specifies the tags for the registered service
	// instance. Multiple tags should be comma separated. Whitespace around
	// the tags is automatically trimmed.
	AnnotationServiceTags = "flomesh.io/service-tags"

	// AnnotationServiceMetaPrefix is the prefix for setting meta key/value
	// for a service. The remainder of the key is the meta key.
	AnnotationServiceMetaPrefix = "flomesh.io/service-meta-"

	// AnnotationServiceWeight is the key of the annotation that determines
	// the traffic weight of the service which is spanned over multiple k8s cluster.
	// e.g. CatalogService `backend` in k8s cluster `A` receives 25% of the traffic
	// compared to same `backend` service in k8s cluster `B`.
	AnnotationServiceWeight = "flomesh.io/service-weight"
)
