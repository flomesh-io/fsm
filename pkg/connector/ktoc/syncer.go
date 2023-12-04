package ktoc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	mapset "github.com/deckarep/golang-set"

	"github.com/flomesh-io/fsm/pkg/connector/provider"
)

const (
	// SyncPeriod is how often the syncer will attempt to
	// reconcile the expected service states with the remote cloud server.
	SyncPeriod = 5 * time.Second

	// ServicePollPeriod is how often a service is checked for
	// whether it has instances to reap.
	ServicePollPeriod = 10 * time.Second
)

// Syncer is responsible for syncing a set of cloud catalog registrations.
// An external system manages the set of registrations and periodically
// updates the Syncer. The Syncer should keep the remote system in sync with
// the given set of registrations.
type Syncer interface {
	// Sync is called to sync the full set of registrations.
	Sync([]*provider.CatalogRegistration)
}

// CloudSyncer is a Syncer that takes the set of registrations and
// registers them with cloud. It also watches cloud for changes to the
// services and ensures the local set of registrations represents the
// source of truth, overwriting any external changes to the services.
type CloudSyncer struct {
	// EnableNamespaces indicates that a user is running Consul Enterprise
	// with version 1.7+ which is namespace aware. It enables Consul namespaces,
	// with syncing into either a single Consul namespace or mirrored from
	// k8s namespaces.
	EnableNamespaces bool

	// CrossNamespaceACLPolicy is the name of the ACL policy to attach to
	// any created Consul namespaces to allow cross namespace service discovery.
	// Only necessary if ACLs are enabled.
	CrossNamespaceACLPolicy string

	// SyncPeriod is the interval between full catalog syncs. These will
	// re-register all services to prevent overwrites of data. This should
	// happen relatively infrequently and default to 5 seconds.
	//
	// ServicePollPeriod is the interval to look for invalid services to
	// deregister. One request will be made for each synced service in
	// Kubernetes.
	//
	// For both syncs, smaller more frequent and focused syncs may be
	// triggered by known drift or changes.
	SyncPeriod        time.Duration
	ServicePollPeriod time.Duration

	// ConsulK8STag is the tag value for services registered.
	ConsulK8STag string

	// The Consul node name to register services with.
	ConsulNodeName string

	lock sync.Mutex
	once sync.Once

	// initialSync is used to ensure that we have received our initial list
	// of services before we start reaping services. When it is closed,
	// the initial sync is complete.
	initialSync chan bool
	// initialSyncOnce controls the close operation on the initialSync channel
	// to ensure it isn't closed more than once.
	initialSyncOnce sync.Once

	// serviceNames is all namespaces mapped to a set of valid
	// cloud service names
	serviceNames map[string]mapset.Set

	// namespaces is all namespaces mapped to a map of cloud service
	// ids mapped to their CatalogRegistrations
	namespaces map[string]map[string]*provider.CatalogRegistration
	deregs     map[string]*provider.CatalogDeregistration

	// watchers is all namespaces mapped to a map of cloud service
	// names mapped to a cancel function for watcher routines
	watchers map[string]map[string]context.CancelFunc

	DiscClient provider.ServiceDiscoveryClient
}

// Sync implements Syncer.
func (s *CloudSyncer) Sync(rs []*provider.CatalogRegistration) {
	// Grab the lock so we can replace the sync state
	s.lock.Lock()
	defer s.lock.Unlock()

	s.serviceNames = make(map[string]mapset.Set)
	s.namespaces = make(map[string]map[string]*provider.CatalogRegistration)

	for _, r := range rs {
		// Determine the namespace the service is in to use for indexing
		// against the s.serviceNames and s.namespaces maps.
		// This will be "" for OSS.
		ns := r.Service.Namespace

		// Mark this as a valid service, initializing state if necessary
		if _, ok := s.serviceNames[ns]; !ok {
			s.serviceNames[ns] = mapset.NewSet()
		}
		s.serviceNames[ns].Add(r.Service.Service)
		log.Debug().Msgf("[Sync] adding service to serviceNames set service:%v service name:%s", r.Service, r.Service.Service)

		// Add service to namespaces map, initializing if necessary
		if _, ok := s.namespaces[ns]; !ok {
			s.namespaces[ns] = make(map[string]*provider.CatalogRegistration)
		}
		s.namespaces[ns][r.Service.ID] = r
		log.Debug().Msgf("[Sync] adding service to namespaces map service:%v", r.Service)
	}

	// Signal that the initial sync is complete and our maps have been populated.
	// We can now safely reap untracked services.
	s.initialSyncOnce.Do(func() { close(s.initialSync) })
}

// Run is the long-running runloop for reconciling the local set of
// services to register with the remote state.
func (s *CloudSyncer) Run(ctx context.Context) {
	s.once.Do(s.init)

	// Start the background watchers
	go s.watchReapableServices(ctx)

	reconcileTimer := time.NewTimer(s.SyncPeriod)
	defer reconcileTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("CloudSyncer quitting")
			return

		case <-reconcileTimer.C:
			s.syncFull(ctx)
			reconcileTimer.Reset(s.SyncPeriod)
		}
	}
}

// watchReapableServices is a long-running task started by Run that
// holds blocking queries to the cloud server to watch for any services
// tagged with k8s that are no longer valid and need to be deleted.
// This task only marks them for deletion but doesn't perform the actual
// deletion.
func (s *CloudSyncer) watchReapableServices(ctx context.Context) {
	// We must wait for the initial sync to be complete and our maps to be
	// populated. If we don't wait, we will reap all services tagged with k8s
	// because we have no tracked services in our maps yet.
	<-s.initialSync

	opts := &provider.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   1 * time.Minute,
		Filter:     fmt.Sprintf("\"%s\" in Tags", s.ConsulK8STag),
	}

	if s.EnableNamespaces {
		opts.Namespace = "*"
	}

	// minWait is the minimum time to wait between scheduling service deletes.
	// This prevents a lot of churn in services causing high CPU usage.
	minWait := s.SyncPeriod / 4
	minWaitCh := time.After(0)
	for {
		var err error

		var services *provider.CatalogNodeServiceList
		err = backoff.Retry(func() error {
			services, err = s.DiscClient.NodeServiceList(s.ConsulNodeName, opts)
			return err
		}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))

		if err != nil {
			log.Warn().Msgf("error querying services, will retry err:%v", err)
		} else {
			log.Debug().Msgf("[watchReapableServices] services returned from catalog services:%v",
				services)
		}

		// Wait our minimum time before continuing or retrying
		select {
		case <-minWaitCh:
			if err != nil || services == nil || len(services.Services) == 0 {
				continue
			}

			minWaitCh = time.After(minWait)
		case <-ctx.Done():
			return
		}

		// Lock so we can modify the stored state
		s.lock.Lock()

		// Go through the service array and find services that should be reaped
		for _, service := range services.Services {
			// Check that the namespace exists in the valid service names map
			// before checking whether it contains the service
			svcNs := service.Namespace
			if !s.EnableNamespaces {
				// Set namespace to empty when namespaces are not enabled.
				svcNs = ""
			}
			if _, ok := s.serviceNames[svcNs]; ok {
				// We only care if we don't know about this service at all.
				if s.serviceNames[svcNs].Contains(service.Service) {
					log.Debug().Msgf("[watchReapableServices] serviceNames contains service namespace:%s service-name:%s",
						svcNs,
						service.Service)
					continue
				}
			}

			log.Info().Msgf("invalid service found, scheduling for delete service-name:%s service-id:%s service-namespace:%s",
				service.Service, service.ID, svcNs)
			if err = s.scheduleReapServiceLocked(service.Service, svcNs); err != nil {
				log.Info().Msgf("error querying service for delete service-name:%s service-namespace:%s err:%v",
					service.Service,
					svcNs,
					err)
			}
		}

		s.lock.Unlock()
	}
}

// watchService watches all instances of a service by name for changes
// and schedules re-registration or deletion if necessary.
func (s *CloudSyncer) watchService(ctx context.Context, name, namespace string) {
	log.Info().Msgf("starting service watcher service-name:%s service-namespace:%s", name, namespace)
	defer log.Info().Msgf("stopping service watcher service-name:%s service-namespace:%s", name, namespace)

	for {
		select {
		// Quit if our context is over
		case <-ctx.Done():
			return

		// Wait for our poll period
		case <-time.After(s.SyncPeriod):
		}

		// Set up query options
		queryOpts := &provider.QueryOptions{
			AllowStale: true,
		}
		if s.EnableNamespaces {
			// Sets the Consul namespace to query the catalog
			queryOpts.Namespace = namespace
		}

		var err error
		// Wait for service changes
		var services []*provider.CatalogService
		err = backoff.Retry(func() error {
			services, err = s.DiscClient.CatalogService(name, s.ConsulK8STag, queryOpts)
			return err
		}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
		if err != nil {
			log.Warn().Msgf("error querying service, will retry service-name:%s service-namespace:%s err:%v",
				name,
				namespace, // will be "" if namespaces aren't enabled
				err)
			continue
		}

		// Lock so we can modify the set of actions to take
		s.lock.Lock()

		for _, svc := range services {
			// Make sure the namespace exists before we run checks against it
			if _, ok := s.serviceNames[namespace]; ok {
				// If the service is valid and its info isn't nil, we don't deregister it
				if s.serviceNames[namespace].Contains(svc.ServiceName) && s.namespaces[namespace][svc.ServiceID] != nil {
					continue
				}
			}

			s.deregs[svc.ServiceID] = &provider.CatalogDeregistration{
				Node:      svc.Node,
				ServiceID: svc.ServiceID,
				Service:   svc.ServiceName,
			}
			if s.EnableNamespaces {
				s.deregs[svc.ServiceID].Namespace = namespace
			}
			log.Debug().Msgf("[watchService] service being scheduled for deregistration namespace:%s service name:%s service id:%s service dereg:%v",
				namespace,
				svc.ServiceName,
				svc.ServiceID,
				s.deregs[svc.ServiceID])
		}

		s.lock.Unlock()
	}
}

// scheduleReapService finds all the instances of the service with the given
// name that have the k8s tag and schedules them for removal.
//
// Precondition: lock must be held.
func (s *CloudSyncer) scheduleReapServiceLocked(name, namespace string) error {
	// Set up query options
	opts := provider.QueryOptions{AllowStale: true}
	if s.EnableNamespaces {
		opts.Namespace = namespace
	}

	// Only consider services that are tagged from k8s
	services, err := s.DiscClient.CatalogService(name, s.ConsulK8STag, &opts)
	if err != nil {
		return err
	}

	// Create deregistrations for all of these
	for _, svc := range services {
		s.deregs[svc.ServiceID] = &provider.CatalogDeregistration{
			Node:      svc.Node,
			ServiceID: svc.ServiceID,
			Service:   svc.ServiceName,
		}
		if s.EnableNamespaces {
			s.deregs[svc.ServiceID].Namespace = namespace
		}
		log.Debug().Msgf("[scheduleReapServiceLocked] service being scheduled for deregistration namespace:%s service name:%s service id:%s service dereg:%v",
			namespace,
			svc.ServiceName,
			svc.ServiceID,
			s.deregs[svc.ServiceID])
	}

	return nil
}

// syncFull is called periodically to perform all the write-based API
// calls to sync the data with cloud. This may also start background
// watchers for specific services.
func (s *CloudSyncer) syncFull(ctx context.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Info().Msg("registering services")

	// Update the service watchers
	for ns, watchers := range s.watchers {
		// If the service the watcher is watching is no longer valid,
		// cancel the watcher
		for svc, cf := range watchers {
			if s.serviceNames[ns] == nil || !s.serviceNames[ns].Contains(svc) {
				cf()
				delete(s.watchers[ns], svc)
				log.Debug().Msgf("[syncFull] deleting service watcher namespace:%s service:%s", ns, svc)
			}
		}
	}

	// Start watchers for all services if they're not already running
	for ns, services := range s.serviceNames {
		for svc := range services.Iter() {
			if _, ok := s.watchers[ns][svc.(string)]; !ok {
				svcCtx, cancelF := context.WithCancel(ctx)
				go s.watchService(svcCtx, svc.(string), ns)
				log.Debug().Msgf("[syncFull] starting watchService routine namespace:%s service:%s", ns, svc)

				// Create watcher map if it doesn't exist for this namespace
				if s.watchers[ns] == nil {
					s.watchers[ns] = make(map[string]context.CancelFunc)
				}

				// Add the watcher to our tracking
				s.watchers[ns][svc.(string)] = cancelF
			}
		}
	}

	// Do all deregistrations first.
	for _, r := range s.deregs {
		log.Info().Msgf("deregistering service node-name:%s service-id:%s service-namespace:%s",
			r.Node,
			r.ServiceID,
			r.Namespace)
		err := s.DiscClient.Deregister(r)
		if err != nil {
			log.Warn().Msgf("error deregistering service node-name:%s service-id:%s service-namespace:%s err:%v",
				r.Node,
				r.ServiceID,
				r.Namespace,
				err)
		}
	}

	// Always clear deregistrations, they'll repopulate if we had errors
	s.deregs = make(map[string]*provider.CatalogDeregistration)

	// Register all the services. This will overwrite any changes that
	// may have been made to the registered services.
	for _, services := range s.namespaces {
		for _, r := range services {
			if s.EnableNamespaces {
				_, err := s.DiscClient.EnsureNamespaceExists(r.Service.Namespace, s.CrossNamespaceACLPolicy)
				if err != nil {
					log.Warn().Msgf("error checking and creating Consul namespace node-name:%s service-name:%s consul-namespace-name:%s err:%v",
						r.Node,
						r.Service.Service,
						r.Service.Namespace,
						err)
					continue
				}
			}

			// Register the service.
			err := s.DiscClient.Register(r)
			if err != nil {
				log.Warn().Msgf("error registering service node-name:%s service-name:%s service:%v err:%v",
					r.Node,
					r.Service.Service,
					r.Service,
					err)
				continue
			}

			log.Debug().Msgf("registered service instance node-name:%s service-name:%s namespace-name:%s service:%v",
				r.Node,
				r.Service.Service,
				r.Service.Namespace,
				r.Service)
		}
	}
}

func (s *CloudSyncer) init() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.serviceNames == nil {
		s.serviceNames = make(map[string]mapset.Set)
	}
	if s.namespaces == nil {
		s.namespaces = make(map[string]map[string]*provider.CatalogRegistration)
	}
	if s.deregs == nil {
		s.deregs = make(map[string]*provider.CatalogDeregistration)
	}
	if s.watchers == nil {
		s.watchers = make(map[string]map[string]context.CancelFunc)
	}
	if s.SyncPeriod == 0 {
		s.SyncPeriod = SyncPeriod
	}
	if s.ServicePollPeriod == 0 {
		s.ServicePollPeriod = ServicePollPeriod
	}
	if s.initialSync == nil {
		s.initialSync = make(chan bool)
	}
}
