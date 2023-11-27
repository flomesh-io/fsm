package c2k

import (
	"k8s.io/client-go/tools/cache"
)

// Resource should be implemented by anything that should be watchable
// by Controller. The Resource needs to be aware of how to create the Informer
// that is responsible for making API calls as well as what to do on Upsert
// and Delete.
type Resource interface {
	// Ready wait util ready
	Ready()

	// ServiceInformer returns the SharedIndexInformer that the controller will
	// use to watch for changes. An Informer is the long-running task that
	// holds blocking queries to K8S and stores data in a local store.
	ServiceInformer() cache.SharedIndexInformer

	// EndpointsInformer returns the SharedIndexInformer that the controller will
	// use to watch for changes. An Informer is the long-running task that
	// holds blocking queries to K8S and stores data in a local store.
	EndpointsInformer() cache.SharedIndexInformer

	// GatewayInformer returns the SharedIndexInformer that the controller will
	// use to watch for changes. An Informer is the long-running task that
	// holds blocking queries to K8S and stores data in a local store.
	GatewayInformer() cache.SharedIndexInformer

	// UpsertService is the callback called when processing the queue
	// of changes from the Informer. If an error is returned, the given item
	// will be retried.
	UpsertService(key string, obj interface{}) error

	// DeleteService is called on object deletion.
	// obj is the last known state of the object before deletion. In some
	// cases, it may not be up to date with the latest state of the object.
	// If an error is returned, the given item will be retried.
	DeleteService(key string, obj interface{}) error

	// UpsertEndpoints is the callback called when processing the queue
	// of changes from the Informer. If an error is returned, the given item
	// will be retried.
	UpsertEndpoints(key string, obj interface{}) error

	// DeleteEndpoints is called on object deletion.
	// obj is the last known state of the object before deletion. In some
	// cases, it may not be up to date with the latest state of the object.
	// If an error is returned, the given item will be retried.
	DeleteEndpoints(key string, obj interface{}) error

	// UpsertGateway is the callback called when processing the queue
	// of changes from the Informer. If an error is returned, the given item
	// will be retried.
	UpsertGateway(key string, obj interface{}) error

	// DeleteGateway is called on object deletion.
	// obj is the last known state of the object before deletion. In some
	// cases, it may not be up to date with the latest state of the object.
	// If an error is returned, the given item will be retried.
	DeleteGateway(key string, obj interface{}) error
}

// Backgrounder should be implemented by a Resource that requires additional
// background processing. If a Resource implements this, then the Controller
// will automatically Run the Backgrounder for the duration of the controller.
//
// The channel will be closed when the Controller is quitting. The Controller
// will block until the Backgrounder completes.
type Backgrounder interface {
	Run(<-chan struct{})
}
