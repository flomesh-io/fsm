package connector

const (
	//ConsulDiscoveryService defines consul discovery service name
	ConsulDiscoveryService = "consul"

	//EurekaDiscoveryService defines eureka discovery service name
	EurekaDiscoveryService = "eureka"
)

const (
	// MeshServiceSyncAnnotation defines mesh service sync annotation
	MeshServiceSyncAnnotation = "flomesh.io/mesh-service-sync"

	// CloudServiceInheritedFromAnnotation defines cloud service inherited annotation
	CloudServiceInheritedFromAnnotation = "flomesh.io/cloud-service-inherited-from"

	// MeshEndpointAddrAnnotation defines mesh endpoint addr annotation
	MeshEndpointAddrAnnotation = "flomesh.io/cloud-endpoint-addr"
)
