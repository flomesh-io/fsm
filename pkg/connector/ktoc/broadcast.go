package ktoc

import (
	"sync/atomic"
	"time"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/connector"
)

// BroadcastListener listens for broadcast messages from the message broker
func (t *KtoCSource) BroadcastListener(stopCh <-chan struct{}, syncPeriod time.Duration) {
	// Register for service config updates broadcast by the message broker
	serviceUpdatePubSub := t.msgBroker.GetServiceUpdatePubSub()
	serviceUpdateChan := serviceUpdatePubSub.Sub(announcements.ServiceUpdate.String())
	defer t.msgBroker.Unsub(serviceUpdatePubSub, serviceUpdateChan)

	slidingTimer := time.NewTimer(time.Second * 10)
	defer slidingTimer.Stop()

	lastServiceDetas := uint64(0)

	for {
		select {
		case <-stopCh:
			return
		case <-serviceUpdateChan:
			serviceDetas := atomic.LoadUint64(&t.serviceDetas)
			if lastServiceDetas == serviceDetas {
				atomic.CompareAndSwapUint64(&t.serviceDetas, lastServiceDetas, 0)
			}
		case <-slidingTimer.C:
			serviceDetas := atomic.LoadUint64(&t.serviceDetas)
			if lastServiceDetas != serviceDetas {
				newJob := func() *SyncJob {
					return &SyncJob{
						done:     make(chan struct{}),
						resource: t,
					}
				}
				<-t.msgWorkQueues.AddJob(newJob())
				lastServiceDetas = serviceDetas
			}

			slidingTimer.Reset(syncPeriod)
		}
	}
}

// SyncJob is the job to sync
type SyncJob struct {
	// Optional waiter
	done     chan struct{}
	resource *KtoCSource
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *SyncJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *SyncJob) Run() {
	defer close(job.done)
	t := job.resource
	t.Lock()
	defer t.Unlock()
	// NOTE(mitchellh): This isn't the most efficient way to do this and
	// the times that sync are called are also not the most efficient. All
	// of these are implementation details so lets improve this later when
	// it becomes a performance issue and just do the easy thing first.
	rs := make([]*connector.CatalogRegistration, 0, t.controller.GetK2CContext().RegisteredServiceMap.Count()*4)
	for item := range t.controller.GetK2CContext().RegisteredServiceMap.IterBuffered() {
		if set := item.Val; len(set) > 0 {
			rs = append(rs, set...)
		}
	}
	// Sync, which should be non-blocking in real-world cases
	t.syncer.Sync(rs)
}

// JobName implementation for this job, for logging purposes
func (job *SyncJob) JobName() string {
	return "fsm-connector-ktoc-sync-job"
}
