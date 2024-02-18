package ktoc

import (
	"fmt"
	"time"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
)

// BroadcastListener listens for broadcast messages from the message broker
func (t *ServiceResource) BroadcastListener() {
	// Register for service config updates broadcast by the message broker
	serviceUpdatePubSub := t.MsgBroker.GetServiceUpdatePubSub()
	serviceUpdateChan := serviceUpdatePubSub.Sub(announcements.ServiceUpdate.String())
	defer t.MsgBroker.Unsub(serviceUpdatePubSub, serviceUpdateChan)

	// Wait for two informer synchronization periods
	slidingTimer := time.NewTimer(time.Second * 20)
	defer slidingTimer.Stop()

	reconfirm := true

	for {
		select {
		case <-serviceUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(time.Second * 5)
			// Avoid data omission
			reconfirm = true
		case <-slidingTimer.C:
			newJob := func() *SyncJob {
				return &SyncJob{
					done:     make(chan struct{}),
					resource: t,
				}
			}
			<-t.MsgWorkQueues.AddJob(newJob())

			if reconfirm {
				reconfirm = false
				slidingTimer.Reset(time.Second * 10)
			}
		}
	}
}

// SyncJob is the job to generate pipy policy json
type SyncJob struct {
	// Optional waiter
	done     chan struct{}
	resource *ServiceResource
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *SyncJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *SyncJob) Run() {
	defer close(job.done)
	t := job.resource
	// NOTE(mitchellh): This isn't the most efficient way to do this and
	// the times that sync are called are also not the most efficient. All
	// of these are implementation details so lets improve this later when
	// it becomes a performance issue and just do the easy thing first.
	rs := make([]*provider.CatalogRegistration, 0, len(t.registeredServiceMap)*4)
	for _, set := range t.registeredServiceMap {
		rs = append(rs, set...)
	}
	// Sync, which should be non-blocking in real-world cases
	t.Syncer.Sync(rs)
	fmt.Println("benne fsm connector's job is done.")
}

// JobName implementation for this job, for logging purposes
func (job *SyncJob) JobName() string {
	return "fsm-connector-ktoc-sync-job"
}
