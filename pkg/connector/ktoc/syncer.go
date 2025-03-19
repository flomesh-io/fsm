package ktoc

import (
	"context"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
)

// Syncer is responsible for syncing a set of cloud catalog registrations.
// An external system manages the set of registrations and periodically
// updates the Syncer. The Syncer should keep the remote system in sync with
// the given set of registrations.
type Syncer interface {
	// Sync is called to sync the full set of registrations.
	Sync([]*connector.CatalogRegistration)
}

// KtoCSyncer is a Syncer that takes the set of registrations and
// registers them with cloud. It also watches cloud for changes to the
// services and ensures the local set of registrations represents the
// source of truth, overwriting any external changes to the services.
type KtoCSyncer struct {
	controller connector.ConnectController
	discClient connector.ServiceDiscoveryClient

	lock sync.Mutex
	once sync.Once

	// initialSync is used to ensure that we have received our initial list
	// of services before we start reaping services. When it is closed,
	// the initial sync is complete.
	initialSync chan bool
	// initialSyncOnce controls the close operation on the initialSync channel
	// to ensure it isn't closed more than once.
	initialSyncOnce sync.Once
}

func NewKtoCSyncer(controller connector.ConnectController,
	discClient connector.ServiceDiscoveryClient) *KtoCSyncer {
	return &KtoCSyncer{
		controller: controller,
		discClient: discClient,
	}
}

// Sync implements Syncer.
func (s *KtoCSyncer) Sync(rs []*connector.CatalogRegistration) {
	// Grab the lock so we can replace the sync state
	s.Lock()
	defer s.Unlock()

	s.controller.GetK2CContext().ServiceNames.Clear()
	s.controller.GetK2CContext().Namespaces.Clear()

	for _, r := range rs {
		// Determine the namespace the service is in to use for indexing
		// against the s.serviceNames and s.namespaces maps.
		// This will be "" for OSS.
		ns := r.Service.MicroService.Namespace

		// Mark this as a valid service, initializing state if necessary
		set, ok := s.controller.GetK2CContext().ServiceNames.Get(ns)
		if !ok {
			s.controller.GetK2CContext().ServiceNames.SetIfAbsent(ns, mapset.NewSet())
			set, _ = s.controller.GetK2CContext().ServiceNames.Get(ns)
		}
		set.Add(r.Service.MicroService.Service)
		log.Debug().Msgf("[Sync] adding service to serviceNames set service:%v service name:%s",
			r.Service, r.Service.MicroService.Service)

		// Add service to namespaces map, initializing if necessary
		nsSet, nsOk := s.controller.GetK2CContext().Namespaces.Get(ns)
		if !nsOk {
			s.controller.GetK2CContext().Namespaces.SetIfAbsent(ns, connector.NewConcurrentMap[*connector.CatalogRegistration]())
			nsSet, _ = s.controller.GetK2CContext().Namespaces.Get(ns)
		}
		nsSet.Set(r.Service.ID, r)
		log.Debug().Msgf("[Sync] adding service to namespaces map service:%v", r.Service)
	}

	// Signal that the initial sync is complete and our maps have been populated.
	// We can now safely reap untracked services.
	s.initialSyncOnce.Do(func() {
		go func() {
			time.Sleep(10 * time.Second)
			close(s.initialSync)
		}()
	})
}

// Run is the long-running runloop for reconciling the local set of
// services to register with the remote state.
func (s *KtoCSyncer) Run(ctx context.Context) {
	s.once.Do(s.init)

	// Start the background watchers
	go s.watchReapableServices(ctx)

	reconcileTimer := time.NewTimer(s.controller.GetSyncPeriod())
	defer reconcileTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("KtoCSyncer quitting")
			return

		case <-reconcileTimer.C:
			s.syncFull(ctx)
			reconcileTimer.Reset(s.controller.GetSyncPeriod())
		}
	}
}

// watchReapableServices is a long-running task started by Run that
// holds blocking queries to the cloud server to watch for any services
// tagged with k8s that are no longer valid and need to be deleted.
// This task only marks them for deletion but doesn't perform the actual
// deletion.
func (s *KtoCSyncer) watchReapableServices(ctx context.Context) {
	// We must wait for the initial sync to be complete and our maps to be
	// populated. If we don't wait, we will reap all services tagged with k8s
	// because we have no tracked services in our maps yet.
	<-s.initialSync

	opts := &connector.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   s.controller.GetSyncPeriod(),
	}

	if s.discClient.EnableNamespaces() {
		opts.Namespace = "*"
	}

	// minWait is the minimum time to wait between scheduling service deletes.
	// This prevents a lot of churn in services causing high CPU usage.
	minWait := s.controller.GetSyncPeriod()
	minWaitCh := time.After(minWait)
	var services []ctv1.NamespacedService
	var err error
	for {
		// Wait our minimum time before continuing or retrying
		select {
		case <-minWaitCh:
			services, err = s.discClient.RegisteredServices(opts)
			if err != nil {
				log.Error().Err(err).Msgf("error querying services, will retry")
			} else {
				log.Debug().Msgf("[watchReapableServices] services returned from catalog services:%v", services)
			}

			minWaitCh = time.After(minWait)

			if err != nil || len(services) == 0 {
				continue
			}

		case <-ctx.Done():
			return
		}

		// Lock so we can modify the stored state
		s.Lock()

		// Go through the service array and find services that should be reaped
		for _, service := range services {
			// Check that the namespace exists in the valid service names map
			// before checking whether it contains the service
			svcNs := ""
			if s.discClient.EnableNamespaces() {
				svcNs = service.Namespace
			}
			if set, ok := s.controller.GetK2CContext().ServiceNames.Get(svcNs); ok {
				// We only care if we don't know about this service at all.
				if set.Contains(service.Service) {
					log.Debug().Msgf("[watchReapableServices] serviceNames contains service namespace:%s service-name:%s",
						svcNs,
						service.Service)
					continue
				}
			}

			if err = s.scheduleReapServiceLocked(service.Service, svcNs); err != nil {
				log.Info().Err(err).Msgf("error querying service for delete service-name:%s service-namespace:%s",
					service.Service,
					svcNs)
			}
		}

		s.Unlock()
	}
}

// watchService watches all instances of a service by name for changes
// and schedules re-registration or deletion if necessary.
func (s *KtoCSyncer) watchService(ctx context.Context, name, namespace string) {
	log.Info().Msgf("starting service watcher service-name:%s service-namespace:%s", name, namespace)
	defer log.Info().Msgf("stopping service watcher service-name:%s service-namespace:%s", name, namespace)

	for {
		select {
		// Quit if our context is over
		case <-ctx.Done():
			return

		// Wait for our poll period
		case <-time.After(s.controller.GetSyncPeriod()):
		}

		// Set up query options
		queryOpts := &connector.QueryOptions{
			AllowStale: true,
		}
		if s.discClient.EnableNamespaces() {
			// Sets the Consul namespace to query the catalog
			queryOpts.Namespace = namespace
		}

		instances, err := s.discClient.RegisteredInstances(name, queryOpts)
		if err != nil {
			log.Debug().Err(err).Msgf("error querying service, will retry service-name:%s service-namespace:%s",
				name,
				namespace)
			continue
		}

		// Lock so we can modify the set of actions to take
		s.Lock()

		for _, instance := range instances {
			if len(instance.ServiceID) == 0 {
				continue
			}
			// Make sure the namespace exists before we run checks against it
			if set, ok := s.controller.GetK2CContext().ServiceNames.Get(namespace); ok {
				// If the service is valid and its info isn't nil, we don't deregister it
				if set.Contains(instance.ServiceName) {
					if nsSet, nsOk := s.controller.GetK2CContext().Namespaces.Get(namespace); nsOk {
						if nsSet.Has(instance.ServiceID) {
							continue
						}
					}
				}
			}

			deregistration := &connector.CatalogDeregistration{
				Node:      instance.Node,
				ServiceID: instance.ServiceID,
				NamespacedService: ctv1.NamespacedService{
					Service: instance.ServiceName,
				},
				ServiceRef: instance.ServiceRef,
			}
			if s.discClient.EnableNamespaces() {
				deregistration.Namespace = namespace
			}
			s.controller.GetK2CContext().Deregs.Set(instance.ServiceID, deregistration)
			log.Debug().Msgf("[watchService] service being scheduled for deregistration namespace:%s service name:%s service id:%s service dereg:%v",
				namespace,
				instance.ServiceName,
				instance.ServiceID,
				deregistration)
		}

		s.Unlock()
	}
}

// scheduleReapService finds all the instances of the service with the given
// name that have the k8s tag and schedules them for removal.
//
// Precondition: lock must be held.
func (s *KtoCSyncer) scheduleReapServiceLocked(name, namespace string) error {
	// Set up query options
	opts := connector.QueryOptions{AllowStale: true}
	if s.discClient.EnableNamespaces() {
		opts.Namespace = namespace
	}

	// Only consider services that are tagged from k8s
	instances, err := s.discClient.RegisteredInstances(name, &opts)
	if err != nil {
		return err
	}

	// Create deregistrations for all of these
	for _, instance := range instances {
		if len(instance.ServiceID) == 0 {
			continue
		}
		deregistration := &connector.CatalogDeregistration{
			Node:      instance.Node,
			ServiceID: instance.ServiceID,
			NamespacedService: ctv1.NamespacedService{
				Service: instance.ServiceName,
			},
			ServiceRef: instance.ServiceRef,
		}
		if s.discClient.EnableNamespaces() {
			deregistration.Namespace = namespace
		}
		s.controller.GetK2CContext().Deregs.Set(instance.ServiceID, deregistration)
		log.Debug().Msgf("[scheduleReapServiceLocked] instance being scheduled for deregistration namespace:%s service name:%s service id:%s",
			namespace,
			instance.ServiceName,
			instance.ServiceID)
	}

	return nil
}

// syncFull is called periodically to perform all the write-based API
// calls to sync the data with cloud. This may also start background
// watchers for specific services.
func (s *KtoCSyncer) syncFull(ctx context.Context) {
	s.Lock()
	defer s.Unlock()

	if s.controller.Purge() {
		s.controller.GetK2CContext().ServiceMap.Clear()
		s.controller.GetK2CContext().EndpointsMap.Clear()
		s.controller.GetK2CContext().RegisteredServiceMap.Clear()
	}

	log.Info().Msg("registering services")

	// Update the service watchers
	for witem := range s.controller.GetK2CContext().Watchers.IterBuffered() {
		ns := witem.Key
		watchers := witem.Val
		// If the service the watcher is watching is no longer valid,
		// cancel the watcher
		for item := range watchers.IterBuffered() {
			svc := item.Key
			cancelFunc := item.Val
			if set, ok := s.controller.GetK2CContext().ServiceNames.Get(ns); !ok || !set.Contains(svc) {
				cancelFunc()
				if w, exists := s.controller.GetK2CContext().Watchers.Get(ns); exists {
					w.Remove(svc)
				}
				log.Debug().Msgf("[syncFull] deleting service watcher namespace:%s service:%s", ns, svc)
			}
		}
	}

	// Start watchers for all services if they're not already running
	for item := range s.controller.GetK2CContext().ServiceNames.IterBuffered() {
		ns := item.Key
		services := item.Val
		for svc := range services.Iter() {
			nsWatchers, ok := s.controller.GetK2CContext().Watchers.Get(ns)
			if !ok {
				// Create watcher map if it doesn't exist for this namespace
				s.controller.GetK2CContext().Watchers.SetIfAbsent(ns, connector.NewConcurrentMap[context.CancelFunc]())
				nsWatchers, _ = s.controller.GetK2CContext().Watchers.Get(ns)
			}
			if has := nsWatchers.Has(svc.(string)); !has {
				svcCtx, cancelFunc := context.WithCancel(ctx)
				go s.watchService(svcCtx, svc.(string), ns)
				log.Debug().Msgf("[syncFull] starting watchService routine namespace:%s service:%s", ns, svc)

				// Add the watcher to our tracking
				nsWatchers.Set(svc.(string), cancelFunc)
			}
		}
	}

	deregCnt := 0
	deregWg := new(sync.WaitGroup)
	// Do all deregistrations first.
	for item := range s.controller.GetK2CContext().Deregs.IterBuffered() {
		service := item.Val
		if len(service.ServiceID) == 0 {
			continue
		}
		deregWg.Add(1)
		deregCnt++
		go func(r *connector.CatalogDeregistration) {
			defer deregWg.Done()
			maxRetries := 1
			for maxRetries > 0 {
				log.Info().Msgf("deregistering service service-id:%s service-namespace:%s",
					r.ServiceID,
					r.Namespace)
				if err := s.discClient.Deregister(r); err != nil {
					log.Error().Err(err).Msgf("error deregistering service service-id:%s service-namespace:%s",
						r.ServiceID,
						r.Namespace)
					maxRetries--
					if maxRetries > 0 {
						time.Sleep(time.Second)
					} else {
						break
					}
				} else {
					break
				}
			}
		}(service)
	}
	deregWg.Wait()

	// Always clear deregistrations, they'll repopulate if we had errors
	s.controller.GetK2CContext().Deregs.Clear()

	regCnt := 0
	regWg := new(sync.WaitGroup)
	// Register all the services. This will overwrite any changes that
	// may have been made to the registered services.
	for item := range s.controller.GetK2CContext().Namespaces.IterBuffered() {
		namespace := item.Key
		services := item.Val
		if s.discClient.EnableNamespaces() {
			_, err := s.discClient.EnsureNamespaceExists(namespace)
			if err != nil {
				log.Warn().Msgf("error checking and creating cloud namespace:%s err:%v",
					namespace,
					err)
				continue
			}
		}
		for serviceItem := range services.IterBuffered() {
			service := serviceItem.Val
			regWg.Add(1)
			regCnt++
			go func(r *connector.CatalogRegistration) {
				defer regWg.Done()
				maxRetries := 1
				// Register the service.
				for maxRetries > 0 {
					if err := s.discClient.Register(r); err != nil {
						log.Error().Err(err).Msgf("error registering service service-name:%s",
							r.Service.MicroService.Service)
						maxRetries--
						if maxRetries > 0 {
							time.Sleep(time.Second)
						} else {
							break
						}
					} else {
						log.Debug().Msgf("registered service instance service-name:%s namespace-name:%s",
							r.Service.MicroService.Service,
							r.Service.MicroService.Namespace)
						break
					}
				}
			}(service)
		}
	}
	regWg.Wait()
}

func (s *KtoCSyncer) Lock() {
	s.lock.Lock()
}

func (s *KtoCSyncer) Unlock() {
	s.lock.Unlock()
}

func (s *KtoCSyncer) init() {
	s.Lock()
	defer s.Unlock()
	if s.initialSync == nil {
		s.initialSync = make(chan bool)
	}
}
