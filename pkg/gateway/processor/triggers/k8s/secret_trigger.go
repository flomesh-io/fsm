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

package triggers

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// SecretTrigger is a processor for Secret objects
type SecretTrigger struct{}

// Insert adds a Secret object to the cache and returns true if the cache is changed
func (p *SecretTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(secret)

	return cache.IsSecretReferred(key)
}

// Delete removes a Secret object from the cache and returns true if the cache is changed
func (p *SecretTrigger) Delete(obj interface{}, cache processor.Processor) bool {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(secret)

	return cache.IsSecretReferred(key)
}
