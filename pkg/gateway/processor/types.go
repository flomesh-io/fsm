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

// Package processor contains the processor for the gateway
package processor

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
)

// Trigger is the interface for the functionality provided by the resources
type Trigger interface {
	Insert(obj interface{}, processor Processor) bool
	Delete(obj interface{}, processor Processor) bool
}

// Processor is the interface for the functionality provided by the cache
type Processor interface {
	Insert(obj interface{}) bool
	Delete(obj interface{}) bool
	BuildConfigs()
	IsEffectiveRoute(parentRefs []gwv1.ParentReference) bool
	IsRoutableService(service client.ObjectKey) bool
	IsHeadlessServiceWithoutSelector(key client.ObjectKey) bool
	IsEffectiveTargetRef(policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool
	IsRoutableTargetService(policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) bool
	IsRoutableNamespacedTargetServices(policy client.Object, targetRefs []gwv1alpha2.NamespacedPolicyTargetReference) bool
	IsRoutableLocalTargetServices(policy client.Object, targetRefs []gwv1alpha2.LocalPolicyTargetReference) bool
	IsValidLocalTargetRoutes(policy client.Object, targetRefs []gwpav1alpha2.LocalFilterPolicyTargetReference) bool
	IsConfigMapReferred(cm client.ObjectKey) bool
	IsSecretReferred(secret client.ObjectKey) bool
	IsFilterReferred(filter client.ObjectKey) bool
	IsListenerFilterReferred(filter client.ObjectKey) bool
	IsFilterDefinitionReferred(filter client.ObjectKey) bool
	IsFilterConfigReferred(kind string, config client.ObjectKey) bool
	UseEndpointSlices() bool
}

// Generator is the interface for processing the gateway resources and building the configuration
type Generator interface {
	Generate() fgw.Config
}
