package repo

import (
	"github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/cache"
)

// IngressConfGeneratorJob is the job to generate pipy policy json
type IngressConfGeneratorJob struct {
	cache *cache.Cache

	// Optional waiter
	done chan struct{}
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *IngressConfGeneratorJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *IngressConfGeneratorJob) Run() {
	defer close(job.done)

	job.cache.SyncRoutes()
}

// JobName implementation for this job, for logging purposes
func (job *IngressConfGeneratorJob) JobName() string {
	return "ingress-cfg-job"
}
