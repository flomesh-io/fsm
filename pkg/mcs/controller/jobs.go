package controller

import (
	"github.com/flomesh-io/fsm/pkg/gateway/cache"
)

// ServiceExportCreatedJob is the job to generate pipy policy json
type ServiceExportCreatedJob struct {
	cache cache.Cache

	// Optional waiter
	done chan struct{}
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *ServiceExportCreatedJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *ServiceExportCreatedJob) Run() {
	defer close(job.done)

	job.cache.BuildConfigs()
}

// JobName implementation for this job, for logging purposes
func (job *ServiceExportCreatedJob) JobName() string {
	return "gateway-cfg-job"
}
