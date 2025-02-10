package v2

type xnetworkMeshJob struct {
	done   chan struct{}
	server *Server
}

func (job *xnetworkMeshJob) GetDoneCh() <-chan struct{} {
	return job.done
}

func (job *xnetworkMeshJob) Run() {
	defer close(job.done)
	job.server.doConfigAcls()
}

func (job *xnetworkMeshJob) JobName() string {
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
