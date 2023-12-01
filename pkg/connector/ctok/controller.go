package ctok

import (
	"context"
	"fmt"
	"sync"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("connector-c2k")
)

// Controller is a generic cache.Controller implementation that watches
// Kubernetes for changes to specific set of resources and calls the configured
// callbacks as data changes.
type Controller struct {
	Resource Resource

	servicesInformer  cache.SharedIndexInformer
	endpointsInformer cache.SharedIndexInformer
}

// Event is something that occurred to the resources we're watching.
type Event struct {
	// Key is in the form of <namespace>/<name>, e.g. default/pod-abc123,
	// and corresponds to the resource modified.
	Key string
	// Obj holds the resource that was modified at the time of the event
	// occurring. If possible, the resource should be retrieved from the informer
	// cache, instead of using this field because the cache will be more up to
	// date at the time the event is processed.
	// In some cases, such as a delete event, the resource will no longer exist
	// in the cache and then it is useful to have the resource here.
	Obj interface{}
}

// Run starts the Controller and blocks until stopCh is closed.
//
// Important: Callers must ensure that Run is only called once at a time.
func (c *Controller) Run(stopCh <-chan struct{}) {
	// Properly handle any panics
	defer utilruntime.HandleCrash()

	c.Resource.Ready()

	// Create an informer so we can keep track of all service changes.
	c.servicesInformer = c.Resource.ServiceInformer()

	// Create an informer so we can keep track of all endpoints changes.
	c.endpointsInformer = c.Resource.EndpointsInformer()

	go c.syncService(stopCh)

	go c.syncEndpoints(stopCh)
}

func (c *Controller) syncService(stopCh <-chan struct{}) {
	// Create a queue for storing items to process from the informer.
	var queueOnce sync.Once
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	shutdown := func() { queue.ShutDown() }
	defer queueOnce.Do(shutdown)

	// Add an event handler when data is received from the informer. The
	// event handlers here will block the informer so we just offload them
	// immediately into a workqueue.
	_, err := c.servicesInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// convert the resource object into a key (in this case
			// we are just doing it in the format of 'namespace/name')
			key, err := cache.MetaNamespaceKeyFunc(obj)
			log.Debug().Msgf("queue op add service key:%s", key)
			if err == nil {
				queue.Add(Event{Key: key, Obj: obj})
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			log.Debug().Msgf("queue op update service key:%s", key)
			if err == nil {
				queue.Add(Event{Key: key, Obj: newObj})
			}
		},
		DeleteFunc: c.informerDeleteHandler(queue),
	})
	if err != nil {
		log.Err(err).Msg("error adding service informer event handlers")
	}

	// If the type is a background syncer, then we startup the background
	// process.
	if bg, ok := c.Resource.(Backgrounder); ok {
		ctx, cancelF := context.WithCancel(context.Background())

		// Run the backgrounder
		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			bg.Run(ctx.Done())
		}()

		// Start a goroutine that automatically closes the context when we stop
		go func() {
			select {
			case <-stopCh:
				cancelF()

			case <-ctx.Done():
				// Cancelled outside
			}
		}()

		// When we exit, close the context so the backgrounder ends
		defer func() {
			cancelF()
			<-doneCh
		}()
	}

	// Run the informer to start requesting resources
	go func() {
		c.servicesInformer.Run(stopCh)

		// We have to shut down the queue here if we stop so that
		// wait.Until stops below too. We can't wait until the defer at
		// the top since wait.Until will block.
		queueOnce.Do(shutdown)
	}()

	// Initial sync
	if !cache.WaitForCacheSync(stopCh, c.servicesInformer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("error syncing service cache"))
		return
	}
	log.Debug().Msg("initial service cache sync complete")

	// run the runWorker method every second with a stop channel
	wait.Until(func() {
		for c.processSingleService(queue, c.servicesInformer) {
			// Process
		}
	}, time.Second, stopCh)
}

func (c *Controller) syncEndpoints(stopCh <-chan struct{}) {
	// Create a queue for storing items to process from the informer.
	var queueOnce sync.Once
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	shutdown := func() { queue.ShutDown() }
	defer queueOnce.Do(shutdown)

	// Add an event handler when data is received from the informer. The
	// event handlers here will block the informer so we just offload them
	// immediately into a workqueue.
	_, err := c.endpointsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// convert the resource object into a key (in this case
			// we are just doing it in the format of 'namespace/name')
			key, err := cache.MetaNamespaceKeyFunc(obj)
			log.Debug().Msgf("queue op add endpoints key:%s", key)
			if err == nil {
				queue.Add(Event{Key: key, Obj: obj})
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			log.Debug().Msgf("queue op update endpoints key:%s", key)
			if err == nil {
				queue.Add(Event{Key: key, Obj: newObj})
			}
		},
		DeleteFunc: c.informerDeleteHandler(queue),
	})
	if err != nil {
		log.Err(err).Msg("error adding endpoints informer event handlers")
	}

	// If the type is a background syncer, then we startup the background
	// process.
	if bg, ok := c.Resource.(Backgrounder); ok {
		ctx, cancelF := context.WithCancel(context.Background())

		// Run the backgrounder
		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			bg.Run(ctx.Done())
		}()

		// Start a goroutine that automatically closes the context when we stop
		go func() {
			select {
			case <-stopCh:
				cancelF()

			case <-ctx.Done():
				// Cancelled outside
			}
		}()

		// When we exit, close the context so the backgrounder ends
		defer func() {
			cancelF()
			<-doneCh
		}()
	}

	// Run the informer to start requesting resources
	go func() {
		c.endpointsInformer.Run(stopCh)

		// We have to shut down the queue here if we stop so that
		// wait.Until stops below too. We can't wait until the defer at
		// the top since wait.Until will block.
		queueOnce.Do(shutdown)
	}()

	// Initial sync
	if !cache.WaitForCacheSync(stopCh, c.endpointsInformer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("error syncing endpoints cache"))
		return
	}
	log.Debug().Msg("initial endpoints cache sync complete")

	// run the runWorker method every second with a stop channel
	wait.Until(func() {
		for c.processSingleEndpoints(queue, c.endpointsInformer) {
			// Process
		}
	}, time.Second, stopCh)
}

func (c *Controller) processSingleService(
	queue workqueue.RateLimitingInterface,
	serviceInformer cache.SharedIndexInformer) bool {
	// Fetch the next item
	rawEvent, quit := queue.Get()
	if quit {
		return false
	}
	defer queue.Done(rawEvent)

	event, ok := rawEvent.(Event)
	if !ok {
		log.Warn().Msgf("processSingleService: dropping event with unexpected type, event:%s", rawEvent)
		return true
	}

	// Get the item from the informer to ensure we have the most up-to-date
	// copy.
	key := event.Key
	serviceItem, serviceExists, serviceErr := serviceInformer.GetIndexer().GetByKey(key)

	// If we got the item successfully, call the proper method
	if serviceErr == nil {
		log.Trace().Msgf("processing service object, key:%s exists:%v", key, serviceExists)
		log.Trace().Msgf("processing service object, object:%v", serviceItem)
		if !serviceExists {
			// In the case of deletes, the item is no longer in the cache so
			// we use the copy we got at the time of the event (event.Obj).
			serviceErr = c.Resource.DeleteService(key, event.Obj)
		} else {
			serviceErr = c.Resource.UpsertService(key, serviceItem)
		}

		if serviceErr == nil {
			queue.Forget(rawEvent)
		}
	}

	if serviceErr != nil {
		if queue.NumRequeues(event) < 5 {
			log.Err(serviceErr).Msgf("failed processing service item, retrying key:%s", key)
			queue.AddRateLimited(rawEvent)
		} else {
			log.Err(serviceErr).Msgf("failed processing service item, no more retries, key:%s", key)
			queue.Forget(rawEvent)
			utilruntime.HandleError(serviceErr)
		}
	}

	return true
}

func (c *Controller) processSingleEndpoints(
	queue workqueue.RateLimitingInterface,
	endpointsInformer cache.SharedIndexInformer) bool {
	// Fetch the next item
	rawEvent, quit := queue.Get()
	if quit {
		return false
	}
	defer queue.Done(rawEvent)

	event, ok := rawEvent.(Event)
	if !ok {
		log.Warn().Msgf("processSingleEndpoints: dropping event with unexpected type, event:%s", rawEvent)
		return true
	}

	// Get the item from the informer to ensure we have the most up-to-date
	// copy.
	key := event.Key
	endpointsItem, endpointsExists, endpointsErr := endpointsInformer.GetIndexer().GetByKey(key)

	// If we got the item successfully, call the proper method
	if endpointsErr == nil {
		log.Trace().Msgf("processing endpoints object, key:%s exists:%v", key, endpointsExists)
		log.Trace().Msgf("processing endpoints object, object:%v", endpointsItem)
		if !endpointsExists {
			// In the case of deletes, the item is no longer in the cache so
			// we use the copy we got at the time of the event (event.Obj).
			endpointsErr = c.Resource.DeleteEndpoints(key, event.Obj)
		} else {
			endpointsErr = c.Resource.UpsertEndpoints(key, endpointsItem)
		}

		if endpointsErr == nil {
			queue.Forget(rawEvent)
		}
	}

	if endpointsErr != nil {
		if queue.NumRequeues(event) < 5 {
			log.Warn().Err(endpointsErr).Msgf("failed processing endpoints item, retrying key:%s", key)
			queue.AddRateLimited(rawEvent)
		} else {
			log.Err(endpointsErr).Msgf("failed processing endpoints item, no more retries, key:%s", key)
			queue.Forget(rawEvent)
			utilruntime.HandleError(endpointsErr)
		}
	}

	return true
}

// informerDeleteHandler returns a function that implements
// `DeleteFunc` from the `ResourceEventHandlerFuncs` interface.
// It is split out as its own method to aid in testing.
func (c *Controller) informerDeleteHandler(queue workqueue.RateLimitingInterface) func(obj interface{}) {
	return func(obj interface{}) {
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
		log.Debug().Msgf("queue op delete key:%s", key)
		if err == nil {
			// obj might be of type `cache.DeletedFinalStateUnknown`
			// in which case we need to extract the object from
			// within that struct.
			if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				queue.Add(Event{Key: key, Obj: d.Obj})
			} else {
				queue.Add(Event{Key: key, Obj: obj})
			}
		}
	}
}
