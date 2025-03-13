package ctok

import (
	"context"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
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
	syncer      *CtoKSyncer
	serviceName string
}

// Run is the logic unit of job
func (job *DeleteSyncJob) Run() {
	defer job.wg.Done()
	defer close(job.done)

	_, exists, err := job.syncer.svcInformer.GetIndexer().GetByKey(fmt.Sprintf("%s/%s", job.syncer.namespace(), job.serviceName))
	if err == nil && !exists {
		return
	}

	if err = job.svcClient.Delete(job.ctx, job.serviceName, metav1.DeleteOptions{}); err != nil {
		log.Debug().Msgf("warn deleting service, name:%s warn:%v", job.serviceName, err)
	} else {
		job.syncer.lock.Lock()
		defer job.syncer.lock.Unlock()
		delete(job.syncer.controller.GetC2KContext().SyncedKubeServiceHash, connector.KubeSvcName(job.serviceName))
		delete(job.syncer.controller.GetC2KContext().SyncedKubeServiceCache, connector.KubeSvcName(job.serviceName))
	}
}

// CreateSyncJob is the job to sync
type CreateSyncJob struct {
	*SyncJob
	ctx       context.Context
	wg        *sync.WaitGroup
	syncer    *CtoKSyncer
	svcClient typedcorev1.ServiceInterface
	eptClient typedcorev1.EndpointsInterface
	create    *syncCreate
}

// Run is the logic unit of job
func (job *CreateSyncJob) Run() {
	defer job.wg.Done()
	defer close(job.done)

	key := fmt.Sprintf("%s/%s", job.syncer.namespace(), job.create.service.Name)
	if item, exists, err := job.syncer.svcInformer.GetIndexer().GetByKey(key); err == nil && exists {
		existsService := item.(*corev1.Service)
		preHash := job.syncer.serviceHash(existsService)
		curHash := job.syncer.serviceHash(job.create.service)
		if preHash == curHash {
			return
		} else {
			existsService.Labels = job.create.service.Labels
			existsService.Annotations = job.create.service.Annotations
			existsService.Spec = job.create.service.Spec
			if existsService, err = job.svcClient.Update(job.ctx, existsService, metav1.UpdateOptions{}); err != nil {
				log.Error().Msgf("updating service, name:%s error:%v", job.create.service.Name, err)
				return
			}

			if item, exists, err = job.syncer.eptInformer.GetIndexer().GetByKey(key); err == nil && exists {
				if job.create.endpoints != nil {
					existsEndpoints := item.(*corev1.Endpoints)
					existsEndpoints.Labels = job.create.endpoints.Labels
					existsEndpoints.Annotations = job.create.endpoints.Annotations
					existsEndpoints.Subsets = job.create.endpoints.Subsets
					if _, err = job.eptClient.Update(job.ctx, existsEndpoints, metav1.UpdateOptions{}); err != nil {
						log.Error().Msgf("updating endpoints, name:%s error:%v", job.create.service.Name, err)
					}
				} else {
					if err = job.eptClient.Delete(job.ctx, job.create.service.Name, metav1.DeleteOptions{}); err != nil {
						log.Warn().Msgf("warn deleting endpoints, name:%s error:%v", job.create.service.Name, err)
					}
				}
			} else {
				if job.create.endpoints != nil {
					if _, err := job.eptClient.Create(job.ctx, job.create.endpoints, metav1.CreateOptions{}); err != nil {
						log.Error().Msgf("creating endpoints, name:%s error:%v", job.create.service.Name, err)
					}
				}
			}

			job.syncer.lock.Lock()
			defer job.syncer.lock.Unlock()
			job.syncer.controller.GetC2KContext().SyncedKubeServiceHash[connector.KubeSvcName(job.create.service.Name)] = curHash
			job.syncer.controller.GetC2KContext().SyncedKubeServiceCache[connector.KubeSvcName(job.create.service.Name)] = existsService

			return
		}
	}

	if job.create.endpoints != nil {
		if _, err := job.eptClient.Create(job.ctx, job.create.endpoints, metav1.CreateOptions{}); err != nil {
			log.Error().Msgf("creating endpoints, name:%s error:%v", job.create.service.Name, err)
		}
	}

	if svc, err := job.svcClient.Create(job.ctx, job.create.service, metav1.CreateOptions{}); err != nil {
		log.Error().Msgf("creating service, name:%s error:%v", job.create.service.Name, err)
	} else {
		job.syncer.lock.Lock()
		defer job.syncer.lock.Unlock()
		curHash := job.syncer.serviceHash(svc)
		job.syncer.controller.GetC2KContext().SyncedKubeServiceHash[connector.KubeSvcName(job.create.service.Name)] = curHash
		job.syncer.controller.GetC2KContext().SyncedKubeServiceCache[connector.KubeSvcName(job.create.service.Name)] = svc
	}
}
