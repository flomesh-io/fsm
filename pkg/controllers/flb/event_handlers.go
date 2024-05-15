package flb

import (
	"context"

	"k8s.io/client-go/util/retry"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/flb"
)

func (r *serviceReconciler) onSvcAdd(_ interface{}) {}

func (r *serviceReconciler) onSvcUpdate(oldObj, newObj interface{}) {
	log.Debug().Msgf("[FLB] Service updated")

	oldSvc, ok := oldObj.(*corev1.Service)
	if !ok {
		log.Error().Msgf("Unexpected type: %T", oldObj)
	}

	newSvc, ok := newObj.(*corev1.Service)
	if !ok {
		log.Error().Msgf("Unexpected type: %T", oldObj)
	}

	kubeClient := r.fctx.KubeClient
	if flb.IsFLBEnabled(oldSvc, kubeClient) && !flb.IsFLBEnabled(newSvc, kubeClient) {
		retriableFn := func(err error) bool {
			return err != nil
		}

		delFn := func() error {
			svc := newSvc.DeepCopy()

			if err := r.removeServiceHash(context.Background(), svc); err != nil {
				return err
			}

			svc.Status.LoadBalancer.Ingress = nil
			if err := r.fctx.Status().Update(context.Background(), svc); err != nil {
				return err
			}

			if _, err := r.deleteEntryFromFLB(context.Background(), svc); err != nil {
				return err
			}

			return nil
		}

		err := retry.OnError(retry.DefaultBackoff, retriableFn, delFn)
		if err != nil {
			log.Error().Msgf("Failed to delete entry from FLB: %v", err)
			return
		}

		// Remove the service from the cache if it was deleted from FLB
		delete(r.cache, client.ObjectKeyFromObject(newSvc))
	}
}

func (r *serviceReconciler) onSvcDelete(_ interface{}) {}
