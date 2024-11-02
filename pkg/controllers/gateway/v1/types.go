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

// Package v1 contains controller logic for the Gateway API v1.
package v1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/tools/record"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status/gw"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.NewPretty("gatewayapi-controller/v1")
)

// ---

type GatewaySecretReferenceResolver struct {
	update   *gw.GatewayStatusUpdate
	recorder record.EventRecorder
}

func NewGatewaySecretReferenceResolver(update *gw.GatewayStatusUpdate, recorder record.EventRecorder) *GatewaySecretReferenceResolver {
	return &GatewaySecretReferenceResolver{
		update:   update,
		recorder: recorder,
	}
}

func (r *GatewaySecretReferenceResolver) AddInvalidCertificateRefCondition(obj client.Object, ref gwv1.SecretObjectReference) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.GatewayReasonInvalid), "Unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind)

	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		gwv1.GatewayReasonInvalid,
		fmt.Sprintf("Unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind),
	)
}

func (r *GatewaySecretReferenceResolver) AddRefNotPermittedCondition(obj client.Object, ref gwv1.SecretObjectReference) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.GatewayReasonInvalid), "Reference to Secret %s/%s is not allowed", string(*ref.Namespace), ref.Name)

	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		gwv1.GatewayReasonInvalid,
		fmt.Sprintf("Reference to Secret %s/%s is not allowed", string(*ref.Namespace), ref.Name),
	)
}

func (r *GatewaySecretReferenceResolver) AddRefNotFoundCondition(obj client.Object, key types.NamespacedName) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.GatewayReasonInvalid), "Secret %s not found", key.String())

	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		gwv1.GatewayReasonInvalid,
		fmt.Sprintf("Secret %s not found", key.String()),
	)
}

func (r *GatewaySecretReferenceResolver) AddGetRefErrorCondition(obj client.Object, key types.NamespacedName, err error) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.GatewayReasonInvalid), "Failed to get Secret %s: %s", key.String(), err)

	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		gwv1.GatewayReasonInvalid,
		fmt.Sprintf("Failed to get Secret %s: %s", key.String(), err),
	)
}

func (r *GatewaySecretReferenceResolver) AddRefsResolvedCondition(obj runtime.Object) {
	defer r.recorder.Eventf(obj, corev1.EventTypeNormal, string(gwv1.GatewayReasonAccepted), "Gateway BackendTLS Reference resolved")

	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionTrue,
		gwv1.GatewayReasonAccepted,
		"Gateway BackendTLS Reference resolved",
	)
}

func (r *GatewaySecretReferenceResolver) addCondition(conditionType gwv1.GatewayConditionType, status metav1.ConditionStatus, reason gwv1.GatewayConditionReason, message string) {
	r.update.AddCondition(
		conditionType,
		status,
		reason,
		message,
	)
}

// ---

type GatewayListenerSecretReferenceConditionProvider struct {
	update       *gw.GatewayStatusUpdate
	listenerName string
	recorder     record.EventRecorder
}

func NewGatewayListenerSecretReferenceConditionProvider(name string, update *gw.GatewayStatusUpdate, recorder record.EventRecorder) *GatewayListenerSecretReferenceConditionProvider {
	return &GatewayListenerSecretReferenceConditionProvider{
		update:       update,
		listenerName: name,
		recorder:     recorder,
	}
}

func (r *GatewayListenerSecretReferenceConditionProvider) AddInvalidCertificateRefCondition(obj client.Object, ref gwv1.SecretObjectReference) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonInvalidCertificateRef), "Unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind)

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind),
	)
}

func (r *GatewayListenerSecretReferenceConditionProvider) AddRefNotPermittedCondition(obj client.Object, ref gwv1.SecretObjectReference) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonRefNotPermitted), "Reference to Secret %s/%s is not allowed", string(*ref.Namespace), ref.Name)

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonRefNotPermitted,
		fmt.Sprintf("Reference to Secret %s/%s is not allowed", string(*ref.Namespace), ref.Name),
	)
}

func (r *GatewayListenerSecretReferenceConditionProvider) AddRefNotFoundCondition(obj client.Object, key types.NamespacedName) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonInvalidCertificateRef), "Secret %s not found", key.String())

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Secret %s not found", key.String()),
	)
}

func (r *GatewayListenerSecretReferenceConditionProvider) AddGetRefErrorCondition(obj client.Object, key types.NamespacedName, err error) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonInvalidCertificateRef), "Failed to get Secret %s: %s", key.String(), err)

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Failed to get Secret %s: %s", key.String(), err),
	)
}

func (r *GatewayListenerSecretReferenceConditionProvider) AddRefsResolvedCondition(obj runtime.Object) {
	defer r.recorder.Eventf(obj, corev1.EventTypeNormal, string(gwv1.ListenerReasonResolvedRefs), "Secret references of listener %q resolved", r.listenerName)

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionTrue,
		gwv1.ListenerReasonResolvedRefs,
		fmt.Sprintf("Secret references of listener %q resolved", r.listenerName),
	)
}

func (r *GatewayListenerSecretReferenceConditionProvider) addCondition(conditionType gwv1.ListenerConditionType, status metav1.ConditionStatus, reason gwv1.ListenerConditionReason, message string) {
	r.update.AddListenerCondition(
		r.listenerName,
		conditionType,
		status,
		reason,
		message,
	)
}

// ---

type GatewayListenerObjectReferenceConditionProvider struct {
	listenerName string
	update       *gw.GatewayStatusUpdate
	recorder     record.EventRecorder
}

func NewGatewayListenerObjectReferenceConditionProvider(name string, update *gw.GatewayStatusUpdate, recorder record.EventRecorder) *GatewayListenerObjectReferenceConditionProvider {
	return &GatewayListenerObjectReferenceConditionProvider{
		listenerName: name,
		update:       update,
		recorder:     recorder,
	}
}

func (r *GatewayListenerObjectReferenceConditionProvider) AddInvalidRefCondition(obj client.Object, ref gwv1.ObjectReference) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonInvalidCertificateRef), "Unsupported group %s and kind %s for CA Certificate", ref.Group, ref.Kind)

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Unsupported group %s and kind %s for CA Certificate", ref.Group, ref.Kind),
	)
}

func (r *GatewayListenerObjectReferenceConditionProvider) AddRefNotPermittedCondition(obj client.Object, ref gwv1.ObjectReference) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonRefNotPermitted), "Reference to %s %s/%s is not allowed", ref.Kind, gwutils.NamespaceDerefOr(ref.Namespace, obj.GetNamespace()), ref.Name)

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonRefNotPermitted,
		fmt.Sprintf("Reference to %s %s/%s is not allowed", ref.Kind, string(*ref.Namespace), ref.Name),
	)
}

func (r *GatewayListenerObjectReferenceConditionProvider) AddRefNotFoundCondition(obj client.Object, key types.NamespacedName, kind string) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonInvalidCertificateRef), "%s %s not found", kind, key.String())

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("%s %s not found", kind, key.String()),
	)
}

func (r *GatewayListenerObjectReferenceConditionProvider) AddGetRefErrorCondition(obj client.Object, key types.NamespacedName, kind string, err error) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonInvalidCertificateRef), "Failed to get %s %s: %s", kind, key.String(), err)

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Failed to get %s %s: %s", kind, key.String(), err),
	)
}

func (r *GatewayListenerObjectReferenceConditionProvider) AddNoRequiredCAFileCondition(obj client.Object, key types.NamespacedName, kind string) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonInvalidCertificateRef), "No required CA with key %s in %s %s", corev1.ServiceAccountRootCAKey, kind, key.String())

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("No required CA with key %s in %s %s", corev1.ServiceAccountRootCAKey, kind, key.String()),
	)
}

func (r *GatewayListenerObjectReferenceConditionProvider) AddEmptyCACondition(obj client.Object, ref gwv1.ObjectReference) {
	defer r.recorder.Eventf(obj, corev1.EventTypeWarning, string(gwv1.ListenerReasonInvalidCertificateRef), "CA Certificate is empty in %s %s/%s", ref.Kind, gwutils.NamespaceDerefOr(ref.Namespace, obj.GetNamespace()), ref.Name)

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("CA Certificate is empty in %s %s/%s", ref.Kind, gwutils.NamespaceDerefOr(ref.Namespace, obj.GetNamespace()), ref.Name),
	)
}

func (r *GatewayListenerObjectReferenceConditionProvider) AddRefsResolvedCondition(obj runtime.Object) {
	defer r.recorder.Eventf(obj, corev1.EventTypeNormal, string(gwv1.ListenerReasonResolvedRefs), "Object references of all listeners resolved")

	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionTrue,
		gwv1.ListenerReasonResolvedRefs,
		"Object references of all listeners resolved",
	)
}

func (r *GatewayListenerObjectReferenceConditionProvider) addCondition(conditionType gwv1.ListenerConditionType, status metav1.ConditionStatus, reason gwv1.ListenerConditionReason, message string) {
	r.update.AddListenerCondition(
		r.listenerName,
		conditionType,
		status,
		reason,
		message,
	)
}
