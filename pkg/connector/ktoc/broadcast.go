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

	if !slidingWindowEnabled {
		t.syncImmediate(stopCh, serviceUpdateChan)
		return
	}

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
			t.doSync(&lastServiceDetas)
			slidingTimer.Reset(syncPeriod)
		}
	}
}

func (t *KtoCSource) doSync(lastServiceDetas *uint64) {
	serviceDetas := atomic.LoadUint64(&t.serviceDetas)
	if *lastServiceDetas != serviceDetas {
		newJob := func() *SyncJob {
			return &SyncJob{
				done:     make(chan struct{}),
				resource: t,
			}
		}
		<-t.msgWorkQueues.AddJob(newJob())
		*lastServiceDetas = serviceDetas
	}
}

func (t *KtoCSource) syncImmediate(stopCh <-chan struct{}, serviceUpdateChan <-chan interface{}) {
	immediateRegister := !t.controller.GetNacosK2CReconcileTimerEnabled()

	for {
		select {
		case <-stopCh:
			return
		case <-serviceUpdateChan:
			for {
				select {
				case <-serviceUpdateChan:
				default:
					goto process
				}
			}
		process:
			t.Lock()
			ctx := t.controller.GetK2CContext()
			// Collect full registrations for Sync (updates syncer state)
			rs := make([]*connector.CatalogRegistration, 0, ctx.RegisteredServiceMap.Count()*4)
			var dirty []*connector.CatalogRegistration
			if ctx.DirtyKeys.Cardinality() > 0 {
				for _, key := range ctx.DirtyKeys.ToSlice() {
					if regs, ok := ctx.RegisteredServiceMap.Get(key.(string)); ok {
						dirty = append(dirty, regs...)
					}
				}
				ctx.DirtyKeys.Clear()
			}
			for item := range ctx.RegisteredServiceMap.IterBuffered() {
				if set := item.Val; len(set) > 0 {
					rs = append(rs, set...)
				}
			}
			t.syncer.Sync(rs)
			t.Unlock()

			if immediateRegister {
				if len(dirty) > 0 || !ctx.Deregs.IsEmpty() {
					t.syncer.SyncIncremental(dirty)
				} else {
					t.syncer.SyncFull(context.Background())
				}
			}
		}
	}
}

// SyncJob is the job to sync
type SyncJob struct {
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
	rs := make([]*connector.CatalogRegistration, 0, t.controller.GetK2CContext().RegisteredServiceMap.Count()*4)
	for item := range t.controller.GetK2CContext().RegisteredServiceMap.IterBuffered() {
		if set := item.Val; len(set) > 0 {
			rs = append(rs, set...)
		}
	}
	t.syncer.Sync(rs)
}

// JobName implementation for this job, for logging purposes
func (job *SyncJob) JobName() string {
	return "fsm-connector-ktoc-sync-job"
}
