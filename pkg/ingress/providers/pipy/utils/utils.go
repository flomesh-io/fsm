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

// Package utils provides utility functions for the pipy ingress controller
package utils

import (
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("fsm-ingress-utils")
)

// IsValidPipyIngress checks if the ingress is a valid pipy ingress
func IsValidPipyIngress(ing *networkingv1.Ingress) bool {
	// 1. with annotation or IngressClass
	ingressClass, ok := ing.GetAnnotations()[constants.IngressAnnotationKey]
	if !ok && ing.Spec.IngressClassName != nil {
		ingressClass = *ing.Spec.IngressClassName
	}

	defaultClass := constants.DefaultIngressClass
	log.Info().Msgf("IngressClassName/IngressAnnotation = %s", ingressClass)
	log.Info().Msgf("DefaultIngressClass = %s, and IngressPipyClass = %s", defaultClass, constants.IngressPipyClass)

	// 2. empty IngressClass, and pipy is the default IngressClass or no default at all
	if len(ingressClass) == 0 && (defaultClass == constants.IngressPipyClass || len(defaultClass) == 0) {
		return true
	}

	// 3. with IngressClass
	return ingressClass == constants.IngressPipyClass
}

// MetaNamespaceKey returns the key for an object that implements meta.Interface
func MetaNamespaceKey(obj interface{}) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Warn().Err(err)
	}

	return key
}
