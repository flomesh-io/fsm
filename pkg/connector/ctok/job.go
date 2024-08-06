package ctok

import (
	"context"
	"sync"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// SyncJob is the job to sync
type SyncJob struct {
	// Optional waiter
	done chan struct{}
}

// JobName implementation for this job, for logging purposes
func (job *SyncJob) JobName() string {
	return "fsm-connector-ctok-sync-job"
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *SyncJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// DeleteSyncJob is the job to sync
type DeleteSyncJob struct {
	*SyncJob
	ctx         context.Context
	svcClient   typedcorev1.ServiceInterface
	wg          *sync.WaitGroup
	serviceName string
}

// Run is the logic unit of job
func (job *DeleteSyncJob) Run() {
	defer job.wg.Done()
	defer close(job.done)

	if err := job.svcClient.Delete(job.ctx, job.serviceName, metav1.DeleteOptions{}); err != nil {
		log.Warn().Msgf("warn deleting service, name:%s warn:%v", job.serviceName, err)
	}
}

// CreateSyncJob is the job to sync
type CreateSyncJob struct {
	*SyncJob
	ctx       context.Context
	wg        *sync.WaitGroup
	svcClient typedcorev1.ServiceInterface
	service   apiv1.Service
}

// Run is the logic unit of job
func (job *CreateSyncJob) Run() {
	defer job.wg.Done()
	defer close(job.done)

	if _, err := job.svcClient.Create(job.ctx, &job.service, metav1.CreateOptions{}); err != nil {
		log.Error().Msgf("creating service, name:%s error:%v", job.service.Name, err)
	}
}