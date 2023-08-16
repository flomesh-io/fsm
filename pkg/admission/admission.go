/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package admission contains admission controller logic
package admission

import (
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/flomesh-io/fsm/pkg/constants"
)

// NewMutatingWebhookConfiguration creates a new MutatingWebhookConfiguration
func NewMutatingWebhookConfiguration(webhooks []admissionregv1.MutatingWebhook, meshName, fsmVersion string) *admissionregv1.MutatingWebhookConfiguration {
	if len(webhooks) == 0 {
		return nil
	}

	return &admissionregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.DefaultMutatingWebhookConfigurationName,
			Labels: map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.FSMAppInstanceLabelKey: meshName,
				constants.FSMAppVersionLabelKey:  fsmVersion,
				constants.AppLabel:               constants.FSMControllerName,
			},
		},
		Webhooks: webhooks,
	}
}

// NewValidatingWebhookConfiguration creates a new ValidatingWebhookConfiguration
func NewValidatingWebhookConfiguration(webhooks []admissionregv1.ValidatingWebhook, meshName, fsmVersion string) *admissionregv1.ValidatingWebhookConfiguration {
	if len(webhooks) == 0 {
		return nil
	}

	return &admissionregv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.DefaultValidatingWebhookConfigurationName,
			Labels: map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.FSMAppInstanceLabelKey: meshName,
				constants.FSMAppVersionLabelKey:  fsmVersion,
				constants.AppLabel:               constants.FSMControllerName,
			},
		},
		Webhooks: webhooks,
	}
}

// NewMutatingWebhook creates a new MutatingWebhook
func NewMutatingWebhook(
	mutatingWebhookName,
	webhookServiceNamespace,
	webhookServiceName,
	webhookPath string,
	caBundle []byte,
	namespaceSelector *metav1.LabelSelector,
	objectSelector *metav1.LabelSelector,
	failurePolicy admissionregv1.FailurePolicyType,
	rules []admissionregv1.RuleWithOperations,
) admissionregv1.MutatingWebhook {
	//failurePolicy := admissionregv1.Fail
	matchPolicy := admissionregv1.Exact
	sideEffect := admissionregv1.SideEffectClassNone

	result := admissionregv1.MutatingWebhook{
		Name: mutatingWebhookName,
		ClientConfig: admissionregv1.WebhookClientConfig{
			Service: &admissionregv1.ServiceReference{
				Namespace: webhookServiceNamespace,
				Name:      webhookServiceName,
				Path:      &webhookPath,
				Port:      pointer.Int32(constants.FSMWebhookPort),
			},
			CABundle: caBundle,
		},
		FailurePolicy:           &failurePolicy,
		MatchPolicy:             &matchPolicy,
		Rules:                   rules,
		SideEffects:             &sideEffect,
		AdmissionReviewVersions: []string{"v1"},
	}

	if namespaceSelector != nil {
		result.NamespaceSelector = namespaceSelector
	}

	if objectSelector != nil {
		result.ObjectSelector = objectSelector
	}

	return result
}

// NewValidatingWebhook creates a new ValidatingWebhook
func NewValidatingWebhook(
	validatingWebhookName,
	webhookServiceNamespace,
	webhookServiceName,
	webhookPath string,
	caBundle []byte,
	namespaceSelector *metav1.LabelSelector,
	objectSelector *metav1.LabelSelector,
	failurePolicy admissionregv1.FailurePolicyType,
	rules []admissionregv1.RuleWithOperations,
) admissionregv1.ValidatingWebhook {
	//failurePolicy := admissionregv1.Fail
	matchPolicy := admissionregv1.Exact
	sideEffect := admissionregv1.SideEffectClassNone

	result := admissionregv1.ValidatingWebhook{
		Name: validatingWebhookName,
		ClientConfig: admissionregv1.WebhookClientConfig{
			Service: &admissionregv1.ServiceReference{
				Namespace: webhookServiceNamespace,
				Name:      webhookServiceName,
				Path:      &webhookPath,
				Port:      pointer.Int32(constants.FSMWebhookPort),
			},
			CABundle: caBundle,
		},
		FailurePolicy:           &failurePolicy,
		MatchPolicy:             &matchPolicy,
		Rules:                   rules,
		SideEffects:             &sideEffect,
		AdmissionReviewVersions: []string{"v1"},
	}

	if namespaceSelector != nil {
		result.NamespaceSelector = namespaceSelector
	}

	if objectSelector != nil {
		result.ObjectSelector = objectSelector
	}

	return result
}

// NewRule creates a new Rule
func NewRule(
	operations []admissionregv1.OperationType,
	apiGroups, apiVersions, resources []string,
) admissionregv1.RuleWithOperations {
	return admissionregv1.RuleWithOperations{
		Operations: operations,
		Rule: admissionregv1.Rule{
			APIGroups:   apiGroups,
			APIVersions: apiVersions,
			Resources:   resources,
		},
	}
}
