package v2

type xnetworkConfigJob struct {
	done   chan struct{}
	server *Server
}

func (job *xnetworkConfigJob) GetDoneCh() <-chan struct{} {
	return job.done
}

func (job *xnetworkConfigJob) Run() {
	defer close(job.done)
	job.server.doConfigAcls()
}

func (job *xnetworkConfigJob) JobName() string {
	return "fsm-xnetwork-config-job"
}

type xnetworkE4lbJob struct {
	done   chan struct{}
	server *Server
}

func (job *xnetworkE4lbJob) GetDoneCh() <-chan struct{} {
	return job.done
}

func (job *xnetworkE4lbJob) Run() {
	defer close(job.done)
	job.server.doConfigE4LBs()
}

func (job *xnetworkE4lbJob) JobName() string {
	return "fsm-xnetwork-e4lb-job"
}
