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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	update *gw.GatewayStatusUpdate
}

func NewGatewaySecretReferenceResolver(update *gw.GatewayStatusUpdate) *GatewaySecretReferenceResolver {
	return &GatewaySecretReferenceResolver{
		update: update,
	}
}

func (r *GatewaySecretReferenceResolver) AddInvalidCertificateRefCondition(ref gwv1.SecretObjectReference) {
	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		gwv1.GatewayReasonInvalid,
		fmt.Sprintf("Unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind),
	)
}

func (r *GatewaySecretReferenceResolver) AddRefNotPermittedCondition(ref gwv1.SecretObjectReference) {
	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		gwv1.GatewayReasonInvalid,
		fmt.Sprintf("Reference to Secret %s/%s is not allowed", string(*ref.Namespace), ref.Name),
	)
}

func (r *GatewaySecretReferenceResolver) AddRefNotFoundCondition(key types.NamespacedName) {
	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		gwv1.GatewayReasonInvalid,
		fmt.Sprintf("Secret %s not found", key.String()),
	)
}

func (r *GatewaySecretReferenceResolver) AddGetRefErrorCondition(key types.NamespacedName, err error) {
	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		gwv1.GatewayReasonInvalid,
		fmt.Sprintf("Failed to get Secret %s: %s", key.String(), err),
	)
}

func (r *GatewaySecretReferenceResolver) AddRefsResolvedCondition() {
	r.addCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionTrue,
		gwv1.GatewayReasonAccepted,
		"BackendTLS Reference resolved",
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

type GatewayListenerSecretReferenceResolver struct {
	update       *gw.GatewayStatusUpdate
	listenerName string
}

func NewGatewayListenerSecretReferenceResolver(name string, update *gw.GatewayStatusUpdate) *GatewayListenerSecretReferenceResolver {
	return &GatewayListenerSecretReferenceResolver{
		update:       update,
		listenerName: name,
	}
}

func (r *GatewayListenerSecretReferenceResolver) AddInvalidCertificateRefCondition(ref gwv1.SecretObjectReference) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind),
	)
}

func (r *GatewayListenerSecretReferenceResolver) AddRefNotPermittedCondition(ref gwv1.SecretObjectReference) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonRefNotPermitted,
		fmt.Sprintf("Reference to Secret %s/%s is not allowed", string(*ref.Namespace), ref.Name),
	)
}

func (r *GatewayListenerSecretReferenceResolver) AddRefNotFoundCondition(key types.NamespacedName) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Secret %s not found", key.String()),
	)
}

func (r *GatewayListenerSecretReferenceResolver) AddGetRefErrorCondition(key types.NamespacedName, err error) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Failed to get Secret %s: %s", key.String(), err),
	)
}

func (r *GatewayListenerSecretReferenceResolver) AddRefsResolvedCondition() {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionTrue,
		gwv1.ListenerReasonResolvedRefs,
		"References resolved",
	)
}

func (r *GatewayListenerSecretReferenceResolver) addCondition(conditionType gwv1.ListenerConditionType, status metav1.ConditionStatus, reason gwv1.ListenerConditionReason, message string) {
	r.update.AddListenerCondition(
		r.listenerName,
		conditionType,
		status,
		reason,
		message,
	)
}

// ---

type GatewayListenerObjectReferenceResolver struct {
	listenerName string
	update       *gw.GatewayStatusUpdate
}

func NewGatewayListenerObjectReferenceResolver(name string, update *gw.GatewayStatusUpdate) *GatewayListenerObjectReferenceResolver {
	return &GatewayListenerObjectReferenceResolver{
		listenerName: name,
		update:       update,
	}
}

func (r *GatewayListenerObjectReferenceResolver) AddInvalidRefCondition(ref gwv1.ObjectReference) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Unsupported group %s and kind %s for CA Certificate", ref.Group, ref.Kind),
	)
}

func (r *GatewayListenerObjectReferenceResolver) AddRefNotPermittedCondition(ref gwv1.ObjectReference) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonRefNotPermitted,
		fmt.Sprintf("Reference to %s %s/%s is not allowed", ref.Kind, string(*ref.Namespace), ref.Name),
	)
}

func (r *GatewayListenerObjectReferenceResolver) AddRefNotFoundCondition(key types.NamespacedName, kind string) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("%s %s not found", kind, key.String()),
	)
}

func (r *GatewayListenerObjectReferenceResolver) AddGetRefErrorCondition(key types.NamespacedName, kind string, err error) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("Failed to get %s %s: %s", kind, key.String(), err),
	)
}

func (r *GatewayListenerObjectReferenceResolver) AddNoRequiredCAFileCondition(key types.NamespacedName, kind string) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("No required CA with key %s in %s %s", corev1.ServiceAccountRootCAKey, kind, key.String()),
	)
}

func (r *GatewayListenerObjectReferenceResolver) AddEmptyCACondition(ref gwv1.ObjectReference, refererNamespace string) {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionFalse,
		gwv1.ListenerReasonInvalidCertificateRef,
		fmt.Sprintf("CA Certificate is empty in %s %s/%s", ref.Kind, gwutils.NamespaceDerefOr(ref.Namespace, refererNamespace), ref.Name),
	)
}

func (r *GatewayListenerObjectReferenceResolver) AddRefsResolvedCondition() {
	r.addCondition(
		gwv1.ListenerConditionResolvedRefs,
		metav1.ConditionTrue,
		gwv1.ListenerReasonResolvedRefs,
		"References resolved",
	)
}

func (r *GatewayListenerObjectReferenceResolver) addCondition(conditionType gwv1.ListenerConditionType, status metav1.ConditionStatus, reason gwv1.ListenerConditionReason, message string) {
	r.update.AddListenerCondition(
		r.listenerName,
		conditionType,
		status,
		reason,
		message,
	)
}
