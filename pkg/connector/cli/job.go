package cli

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *connectorControllerJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *connectorControllerJob) Run() {
	defer close(job.done)
	c := job.connectorController
	c.Refresh()
}

// JobName implementation for this job, for logging purposes
func (job *connectorControllerJob) JobName() string {
	return "fsm-connector-controller-job"
}
