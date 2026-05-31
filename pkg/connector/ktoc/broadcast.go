package ktoc

import (
	"context"
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

	slidingWindowEnabled := t.controller.GetNacosK2CSlidingWindowEnabled()
	var slidingTimerCh <-chan time.Time
	var slidingTimer *time.Timer
	if slidingWindowEnabled {
		slidingTimer = time.NewTimer(time.Second * 10)
		defer slidingTimer.Stop()
		slidingTimerCh = slidingTimer.C
	}

	immediateCh := make(chan struct{}, 1)

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
			if !slidingWindowEnabled {
				select {
				case immediateCh <- struct{}{}:
				default:
				}
			}
		case <-slidingTimerCh:
			t.doSync(&lastServiceDetas)
			slidingTimer.Reset(syncPeriod)
		case <-immediateCh:
			t.doSync(&lastServiceDetas)
		}
	}
}

func (t *KtoCSource) doSync(lastServiceDetas *uint64) {
	serviceDetas := atomic.LoadUint64(&t.serviceDetas)
	if *lastServiceDetas != serviceDetas {
		newJob := func() *SyncJob {
			return &SyncJob{
				done:              make(chan struct{}),
				resource:          t,
				immediateRegister: !t.controller.GetNacosK2CReconcileTimerEnabled(),
			}
		}
		<-t.msgWorkQueues.AddJob(newJob())
		*lastServiceDetas = serviceDetas
	}
}

// SyncJob is the job to sync
type SyncJob struct {
	// Optional waiter
	done     chan struct{}
	resource *KtoCSource
	// immediateRegister when true executes register/deregister immediately
	// instead of waiting for the next reconcileTimer tick.
	immediateRegister bool
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
	rs := make([]*connector.CatalogRegistration, 0, t.controller.GetK2CContext().RegisteredServiceMap.Count()*4)
	for item := range t.controller.GetK2CContext().RegisteredServiceMap.IterBuffered() {
		if set := item.Val; len(set) > 0 {
			rs = append(rs, set...)
		}
	}
	t.syncer.Sync(rs)

	if job.immediateRegister {
		t.Unlock()
		t.syncer.SyncFull(context.Background())
		return
	}
	t.Unlock()
}

// JobName implementation for this job, for logging purposes
func (job *SyncJob) JobName() string {
	return "fsm-connector-ktoc-sync-job"
}
