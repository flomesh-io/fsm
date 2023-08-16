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

package webhook

import (
	"net/http"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	fctx "github.com/flomesh-io/fsm/pkg/context"
)

// Object is the interface for webhook objects
type Object interface {
	RuntimeObject() runtime.Object
}

// Defaulter is the interface for webhook defaulters
type Defaulter interface {
	Object
	SetDefaults(obj interface{})
}

// Validator is the interface for webhook validators
type Validator interface {
	Object
	ValidateCreate(obj interface{}) error
	ValidateUpdate(oldObj, obj interface{}) error
	ValidateDelete(obj interface{}) error
}

// Register is the interface for webhook registers
type Register interface {
	GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook)
	GetHandlers() map[string]http.Handler
}

// RegisterConfig is the configuration for webhook registers
type RegisterConfig struct {
	*fctx.ControllerContext
	WebhookSvcNs   string
	WebhookSvcName string
	CaBundle       []byte
}
