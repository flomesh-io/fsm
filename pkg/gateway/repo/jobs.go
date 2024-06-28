package repo

import "github.com/flomesh-io/fsm/pkg/gateway/processor"

// GatewayConfGeneratorJob is the job to generate pipy policy json
type GatewayConfGeneratorJob struct {
	cache processor.Processor

	// Optional waiter
	done chan struct{}
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *GatewayConfGeneratorJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *GatewayConfGeneratorJob) Run() {
	defer close(job.done)

	job.cache.BuildConfigs()
}

// JobName implementation for this job, for logging purposes
func (job *GatewayConfGeneratorJob) JobName() string {
	return "gateway-cfg-job"
}
