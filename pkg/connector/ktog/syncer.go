package ktog

import (
	"context"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
)

const (
	// SyncPeriod is how often the syncer will attempt to
	// reconcile the expected Service states.
	SyncPeriod = 5 * time.Second

	// ServicePollPeriod is how often a Service is checked for
	// whether it has instances to reap.
	ServicePollPeriod = 10 * time.Second
)

// Syncer is responsible for syncing a set of gateway routes.
type Syncer interface {
	// Sync is called to sync the full set of registrations.
	Sync([]*corev1.Service)
}

// GatewayRouteSyncer is a Syncer that takes the set of gateway routes.
type GatewayRouteSyncer struct {
	SyncPeriod        time.Duration
	ServicePollPeriod time.Duration

	lock sync.Mutex
	once sync.Once

	// initialSync is used to ensure that we have received our initial list
	// of services before we start reaping services. When it is closed,
	// the initial sync is complete.
	initialSync chan bool
	// initialSyncOnce controls the close operation on the initialSync channel
	// to ensure it isn't closed more than once.
	initialSyncOnce sync.Once

	GatewayResource *GatewayResource

	services map[string]*corev1.Service
	deregs   map[string]*corev1.Service
}

// Sync implements Syncer.
func (s *GatewayRouteSyncer) Sync(rs []*corev1.Service) {
	// Grab the lock so we can replace the sync state
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, svc := range s.services {
		shadowSvc := svc
		s.deregs[string(shadowSvc.UID)] = shadowSvc
	}

	s.services = make(map[string]*corev1.Service)

	for _, svc := range rs {
		shadowSvc := svc
		s.services[string(shadowSvc.UID)] = shadowSvc
		delete(s.deregs, string(shadowSvc.UID))
	}

	// Signal that the initial sync is complete and our maps have been populated.
	// We can now safely reap untracked services.
	s.initialSyncOnce.Do(func() { close(s.initialSync) })
}

// Run is the long-running runloop for reconciling the local set of
// services to register with the remote state.
func (s *GatewayRouteSyncer) Run(ctx context.Context, ctrls ...*connector.CacheController) {
	s.once.Do(s.init)

	for _, ctrl := range ctrls {
		for {
			if ctrl.HasSynced() {
				break
			}
			time.Sleep(time.Second)
		}
	}

	reconcileTimer := time.NewTimer(s.SyncPeriod)
	defer reconcileTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("GatewayRouteSyncer quitting")
			return

		case <-reconcileTimer.C:
			s.syncFull(ctx)
			reconcileTimer.Reset(s.SyncPeriod)
		}
	}
}

// syncFull is called periodically to perform all the write-based API
// calls to sync the data with cloud. This may also start background
// watchers for specific services.
func (s *GatewayRouteSyncer) syncFull(ctx context.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, svc := range s.deregs {
		s.GatewayResource.deleteGatewayRoute(svc.Name, svc.Namespace)
	}

	s.deregs = make(map[string]*corev1.Service)

	for _, svc := range s.services {
		s.GatewayResource.updateGatewayRoute(svc)
	}
}

func (s *GatewayRouteSyncer) init() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.services == nil {
		s.services = make(map[string]*corev1.Service)
	}
	if s.deregs == nil {
		s.deregs = make(map[string]*corev1.Service)
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
