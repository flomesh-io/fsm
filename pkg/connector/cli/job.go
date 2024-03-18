package cli

import "github.com/flomesh-io/fsm/pkg/connector"

// connectControllerJob is the job to generate pipy policy json
type connectControllerJob struct {
	// Optional waiter
	done              chan struct{}
	connectController connector.ConnectController
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *connectControllerJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *connectControllerJob) Run() {
	defer close(job.done)
	c := job.connectController
	c.Refresh()
}

// JobName implementation for this job, for logging purposes
func (job *connectControllerJob) JobName() string {
	return "fsm-connector-controller-job"
}
