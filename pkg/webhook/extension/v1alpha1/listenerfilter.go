package v1alpha1

import (
	"context"
	"fmt"

	"github.com/flomesh-io/fsm/pkg/constants"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/flomesh-io/fsm/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"
)

type ListenerFilterWebhook struct {
	webhook.DefaultWebhook
}

func NewListenerFilterWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &ListenerFilterWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&extv1alpha1.ListenerFilter{}).
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

func (r *ListenerFilterWebhook) Default(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*extv1alpha1.ListenerFilter)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	return nil
}

func (r *ListenerFilterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *ListenerFilterWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *ListenerFilterWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	filter, ok := obj.(*extv1alpha1.ListenerFilter)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errs := r.validateSpec(ctx, filter.Spec, field.NewPath("spec"))

	if len(errs) > 0 {
		return warnings, utils.ErrorListToError(errs)
	}

	return nil, nil
}

func (r *ListenerFilterWebhook) validateSpec(ctx context.Context, spec extv1alpha1.ListenerFilterSpec, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	errs = append(errs, r.validateTargetRefs(ctx, spec.TargetRefs, path.Child("targetRefs"))...)
	errs = append(errs, r.validateDefinitionRef(ctx, spec.DefinitionRef, path.Child("definitionRef"))...)
	errs = append(errs, r.validateConfigRef(ctx, spec.ConfigRef, path.Child("configRef"))...)

	return errs
}

func (r *ListenerFilterWebhook) validateTargetRefs(ctx context.Context, refs []extv1alpha1.LocalTargetReferenceWithPort, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	for i, ref := range refs {
		if ref.Group != gwv1.GroupName {
			errs = append(errs, field.Invalid(path.Index(i).Child("group"), ref.Group, fmt.Sprintf("group must be %s", gwv1.GroupName)))
		}

		if ref.Kind != constants.GatewayAPIGatewayKind {
			errs = append(errs, field.Invalid(path.Index(i).Child("kind"), ref.Kind, "kind must be Gateway"))
		}
	}

	return errs
}

func (r *ListenerFilterWebhook) validateDefinitionRef(ctx context.Context, definitionRef gwv1.LocalObjectReference, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	if definitionRef.Group != extv1alpha1.GroupName {
		errs = append(errs, field.Invalid(path.Child("group"), definitionRef.Group, fmt.Sprintf("group must be %s", extv1alpha1.GroupName)))
	}

	if definitionRef.Kind != constants.GatewayAPIExtensionFilterDefinitionKind {
		errs = append(errs, field.Invalid(path.Child("kind"), definitionRef.Kind, fmt.Sprintf("kind must be %s", constants.GatewayAPIExtensionFilterDefinitionKind)))
	}

	return errs
}

func (r *ListenerFilterWebhook) validateConfigRef(ctx context.Context, configRef *gwv1.LocalObjectReference, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	if configRef != nil && configRef.Group != extv1alpha1.GroupName {
		errs = append(errs, field.Invalid(path.Child("group"), configRef.Group, fmt.Sprintf("group must be %s", extv1alpha1.GroupName)))
	}

	return errs
}
