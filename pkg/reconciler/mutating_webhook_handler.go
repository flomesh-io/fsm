package reconciler

import (
	"context"
	reflect "reflect"
	"strings"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
)

// mutatingWebhookEventHandler creates mutating webhook events handlers.
func (c client) mutatingWebhookEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldMwhc := oldObj.(*admissionv1.MutatingWebhookConfiguration)
			newMwhc := newObj.(*admissionv1.MutatingWebhookConfiguration)
			log.Debug().Msgf("mutating webhook update event for %s", newMwhc.Name)
			if !c.isMutatingWebhookUpdated(oldMwhc, newMwhc) {
				return
			}

			c.reconcileMutatingWebhook(oldMwhc, newMwhc)
		},

		DeleteFunc: func(obj interface{}) {
			mwhc := obj.(*admissionv1.MutatingWebhookConfiguration)
			c.addMutatingWebhook(mwhc)
			log.Debug().Msgf("mutating webhook delete event for %s", mwhc.Name)
		},
	}
}

func (c client) reconcileMutatingWebhook(oldMwhc, newMwhc *admissionv1.MutatingWebhookConfiguration) {
	newMwhc.Webhooks = oldMwhc.Webhooks
	newMwhc.Name = oldMwhc.Name
	newMwhc.Labels = oldMwhc.Labels
	if _, err := c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.Background(), newMwhc, metav1.UpdateOptions{}); err != nil {
		// There might be conflicts when multiple injectors try to update the same resource
		// One of the injectors will successfully update the resource, hence conflicts shoud be ignored and not treated as an error
		if !apierrors.IsConflict(err) {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrReconcilingDeletedMutatingWebhook)).
				Msgf("Error updating mutating webhook: %s", newMwhc.Name)
		}
	}
	metricsstore.DefaultMetricsStore.ReconciliationTotal.WithLabelValues("MutatingWebhookConfiguration").Inc()
	log.Debug().Msgf("Successfully reconciled mutating webhook %s", newMwhc.Name)
}

func (c client) addMutatingWebhook(oldMwhc *admissionv1.MutatingWebhookConfiguration) {
	oldMwhc.ResourceVersion = ""
	if _, err := c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), oldMwhc, metav1.CreateOptions{}); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrReconcilingDeletedMutatingWebhook)).
			Msgf("Error adding back deleted mutating webhook: %s", oldMwhc.Name)
	}
	metricsstore.DefaultMetricsStore.ReconciliationTotal.WithLabelValues("MutatingWebhookConfiguration").Inc()
	log.Debug().Msgf("Successfully added back mutating webhook %s", oldMwhc.Name)
}

func (c *client) isMutatingWebhookUpdated(oldMwhc, newMwhc *admissionv1.MutatingWebhookConfiguration) bool {
	webhookEqual := reflect.DeepEqual(oldMwhc.Webhooks, newMwhc.Webhooks)
	mwhcNameChanged := strings.Compare(oldMwhc.Name, newMwhc.Name) != 0
	mwhcLabelsChanged := isLabelModified(constants.AppLabel, constants.FSMInjectorName, newMwhc.Labels) ||
		isLabelModified(constants.FSMAppVersionLabelKey, c.fsmVersion, newMwhc.Labels)
	mwhcUpdated := !webhookEqual || mwhcNameChanged || mwhcLabelsChanged
	return mwhcUpdated
}
