package ingress

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"github.com/flomesh-io/fsm/pkg/webhook"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type GlobalTrafficPolicyWebhook struct {
	webhook.DefaultWebhook
}

func NewGlobalTrafficPolicyWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &GlobalTrafficPolicyWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&mcsv1alpha1.GlobalTrafficPolicy{}).
		WithWebhookServiceName(cfg.WebhookSvcName).
		WithWebhookServiceNamespace(cfg.WebhookSvcNs).
		WithCABundle(cfg.CaBundle).
		Complete(); err != nil {
		return nil
	} else {
		r.CfgBuilder = blder
	}

	return r
}

func (r *GlobalTrafficPolicyWebhook) Default(_ context.Context, obj runtime.Object) error {
	policy, ok := obj.(*mcsv1alpha1.GlobalTrafficPolicy)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	if policy.Spec.LbType == "" {
		policy.Spec.LbType = mcsv1alpha1.LocalityLbType
	}

	return nil
}

func (r *GlobalTrafficPolicyWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *GlobalTrafficPolicyWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *GlobalTrafficPolicyWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	policy, ok := obj.(*mcsv1alpha1.GlobalTrafficPolicy)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	switch policy.Spec.LbType {
	case mcsv1alpha1.LocalityLbType:
		if len(policy.Spec.Targets) > 1 {
			return nil, fmt.Errorf("in case of Locality load balancer, the traffic can only be sticky to exact one cluster, either in cluster or a specific remote cluster")
		}
	case mcsv1alpha1.FailOverLbType:
		if len(policy.Spec.Targets) == 0 {
			return nil, fmt.Errorf("requires at least one cluster for failover")
		}
	case mcsv1alpha1.ActiveActiveLbType:
		//if len(policy.Spec.Targets) == 0 {
		//	return fmt.Errorf("requires at least another one cluster for active-active load balancing")
		//}

		for _, t := range policy.Spec.Targets {
			if t.Weight != nil && *t.Weight < 0 {
				return nil, fmt.Errorf("weight %d of %s is invalid for active-active load balancing, it must be >= 0", t.Weight, t.ClusterKey)
			}
		}
	}

	return nil, nil
}
