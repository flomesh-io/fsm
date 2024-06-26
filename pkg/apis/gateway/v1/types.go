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

// Package v1 contains the extended GatewayAPI types
package v1

import gwv1 "sigs.k8s.io/gateway-api/apis/v1"

const (
	// GatewayClassConditionStatusActive is the status condition for active GatewayClass
	GatewayClassConditionStatusActive gwv1.GatewayClassConditionType = "Active"

	// GatewayClassReasonActive is the reason for active GatewayClass
	GatewayClassReasonActive gwv1.GatewayClassConditionReason = "Active"

	// GatewayClassReasonInactive is the reason for inactive GatewayClass
	GatewayClassReasonInactive gwv1.GatewayClassConditionReason = "Inactive"
)

const (
	// GatewayConditionEffective is the status condition for active Gateway
	// The active gateway is the oldest gateway in the namespace.
	// If CreationTimestamp is equal, then sort by name in alphabet order.
	GatewayConditionEffective gwv1.GatewayConditionType = "Effective"

	// GatewayReasonEffective is the reason for Effective Gateway
	GatewayReasonEffective gwv1.GatewayConditionReason = "Effective"

	// GatewayReasonNotOldest is the reason for inactive Gateway
	GatewayReasonNotOldest gwv1.GatewayConditionReason = "NotOldest"
)
