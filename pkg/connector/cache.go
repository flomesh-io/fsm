// Package connector contains a reusable abstraction for efficiently
// watching for changes in resources in a Kubernetes cluster.
package connector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// CacheController is a generic cache.Controller implementation that watches
// Kubernetes for changes to specific set of resources and calls the configured
// callbacks as data changes.
type CacheController struct {
	Resource Resource
	informer cache.SharedIndexInformer
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

// Run starts the CacheController and blocks until stopCh is closed.
//
// Important: Callers must ensure that Run is only called once at a time.
func (c *CacheController) Run(stopCh <-chan struct{}) {
	// Properly handle any panics
	defer utilruntime.HandleCrash()

	// Create an informer so we can keep track of all Service changes.
	informer := c.Resource.Informer()
	c.informer = informer

	rateLimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 30*time.Second),
		// 256 qps, 512 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(256), 512)},
	)

	// Create a queue for storing items to process from the informer.
	var queueOnce sync.Once
	queue := workqueue.NewRateLimitingQueue(rateLimiter)
	shutdown := func() { queue.ShutDown() }
	defer queueOnce.Do(shutdown)

	// Add an event handler when data is received from the informer. The
	// event handlers here will block the informer so we just offload them
	// immediately into a workqueue.
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// convert the resource object into a key (in this case
			// we are just doing it in the format of 'namespace/name')
			key, err := cache.MetaNamespaceKeyFunc(obj)
			log.Debug().Msgf("queue op %s %s:%s", "add", "key", key)
			if err == nil {
				queue.Add(Event{Key: key, Obj: obj})
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			log.Debug().Msgf("queue op %s %s:%s", "update", "key", key)
			if err == nil {
				queue.Add(Event{Key: key, Obj: newObj})
			}
		},
		DeleteFunc: c.informerDeleteHandler(queue),
	})
	if err != nil {
		log.Error().Msgf("error adding informer event handlers:%v", err)
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
				return
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
		informer.Run(stopCh)

		// We have to shut down the queue here if we stop so that
		// wait.Until stops below too. We can't wait until the defer at
		// the top since wait.Until will block.
		queueOnce.Do(shutdown)
	}()

	// Initial sync
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("error syncing cache"))
		return
	}
	log.Debug().Msg("initial cache sync complete")

	// run the runWorker method every second with a stop channel
	wait.Until(func() {
		for c.processSingle(queue, informer) {
			// Process
		}
	}, time.Second, stopCh)
}

// HasSynced implements cache.Controller.
func (c *CacheController) HasSynced() bool {
	if c.informer == nil {
		return false
	}

	return c.informer.HasSynced()
}

// LastSyncResourceVersion implements cache.Controller.
func (c *CacheController) LastSyncResourceVersion() string {
	if c.informer == nil {
		return ""
	}

	return c.informer.LastSyncResourceVersion()
}

func (c *CacheController) processSingle(queue workqueue.RateLimitingInterface, informer cache.SharedIndexInformer) bool {
	// Fetch the next item
	rawEvent, quit := queue.Get()
	if quit {
		return false
	}
	defer queue.Done(rawEvent)

	event, ok := rawEvent.(Event)
	if !ok {
		log.Warn().Msgf("processSingle: dropping event with unexpected type event:%v", rawEvent)
		return true
	}

	// Get the item from the informer to ensure we have the most up-to-date
	// copy.
	key := event.Key
	item, exists, err := informer.GetIndexer().GetByKey(key)

	// If we got the item successfully, call the proper method
	if err == nil {
		log.Debug().Msgf("processing object key:%s exists:%v", key, exists)
		log.Trace().Msgf("processing object:%v", item)
		if !exists {
			// In the case of deletes, the item is no longer in the cache so
			// we use the copy we got at the time of the event (event.Obj).
			err = c.Resource.Delete(key, event.Obj)
		} else {
			err = c.Resource.Upsert(key, item)
		}

		if err == nil {
			queue.Forget(rawEvent)
		}
	}

	if err != nil {
		if queue.NumRequeues(event) < 5 {
			log.Debug().Msgf("failed processing item, retrying key:%s error:%v", key, err)
			queue.AddRateLimited(rawEvent)
		} else {
			log.Debug().Msgf("failed processing item, no more retries key:%s error:%v", key, err)
			queue.Forget(rawEvent)
			//utilruntime.HandleError(err)
		}
	}

	return true
}

// informerDeleteHandler returns a function that implements
// `DeleteFunc` from the `ResourceEventHandlerFuncs` interface.
// It is split out as its own method to aid in testing.
func (c *CacheController) informerDeleteHandler(queue workqueue.RateLimitingInterface) func(obj interface{}) {
	return func(obj interface{}) {
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
		log.Debug().Msgf("queue op %s %s:%s", "delete", "key", key)
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
