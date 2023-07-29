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

package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	corev1 "k8s.io/api/core/v1"
)

// SecretProcessor is a processor for Secret objects
type SecretProcessor struct {
}

// Insert adds a Secret object to the cache and returns true if the cache is changed
func (p *SecretProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(secret)
	cache.secrets[key] = struct{}{}

	return cache.isSecretReferredByAnyGateway(key)
}

// Delete removes a Secret object from the cache and returns true if the cache is changed
func (p *SecretProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(secret)
	_, found := cache.secrets[key]
	delete(cache.secrets, key)

	return found
}
