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

package flb

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// IsFLBEnabled checks if the service is enabled for flb
func IsFLBEnabled(svc *corev1.Service, kubeClient kubernetes.Interface) bool {
	if svc == nil {
		return false
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return false
	}

	// if service doesn't have flb.flomesh.io/enabled annotation
	if svc.Annotations == nil || svc.Annotations[constants.FLBEnabledAnnotation] == "" {
		// check ns annotation
		ns, err := kubeClient.CoreV1().
			Namespaces().
			Get(context.TODO(), svc.Namespace, metav1.GetOptions{})

		if err != nil {
			log.Error().Msgf("Failed to get namespace %q: %s", svc.Namespace, err)
			return false
		}

		if ns.Annotations == nil || ns.Annotations[constants.FLBEnabledAnnotation] == "" {
			return false
		}

		log.Debug().Msgf("Found annotation %q on Namespace %q", constants.FLBEnabledAnnotation, ns.Name)
		return utils.ParseEnabled(ns.Annotations[constants.FLBEnabledAnnotation])
	}

	// parse svc annotation
	log.Debug().Msgf("Found annotation %q on Service %s/%s", constants.FLBEnabledAnnotation, svc.Namespace, svc.Name)
	return utils.ParseEnabled(svc.Annotations[constants.FLBEnabledAnnotation])
}

// IsServiceRefToValidTLSSecret checks if the service is referencing to a valid TLS Secret
func IsServiceRefToValidTLSSecret(svc *corev1.Service, kubeClient kubernetes.Interface) (bool, error) {
	if len(svc.Annotations) == 0 {
		return false, fmt.Errorf("service has empty annotations")
	}

	name, ok := svc.Annotations[constants.FLBTLSSecretAnnotation]
	if !ok {
		return false, fmt.Errorf("service doesn't have annotation %s", constants.FLBTLSSecretAnnotation)
	}

	mode := GetTLSSecretMode(svc)
	switch mode {
	case TLSSecretModeRemote:
		return true, nil
	case TLSSecretModeLocal:
		secret, err := kubeClient.CoreV1().Secrets(svc.Namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if !IsFLBTLSSecret(secret) {
			return false, fmt.Errorf("invalid secret, doesn't have required label: %s=true", constants.FLBTLSSecretLabel)
		}

		return IsValidTLSSecret(secret)
	default:
		return false, fmt.Errorf("invalid TLS Secret Mode: %s", mode)
	}
}

// IsValidTLSSecret checks if the secret has a valid TLS Cert and Key
func IsValidTLSSecret(secret *corev1.Secret) (bool, error) {
	cert, ok := secret.Data[corev1.TLSCertKey]
	if !ok {
		return false, fmt.Errorf("secret doesn't have required cert with name %s", corev1.TLSCertKey)
	}

	certBlock, _ := pem.Decode(cert)
	if certBlock == nil {
		return false, fmt.Errorf("failed to parse certificate PEM")
	}

	if _, err := x509.ParseCertificate(certBlock.Bytes); err != nil {
		return false, err
	}

	key, ok := secret.Data[corev1.TLSPrivateKeyKey]
	if !ok {
		return false, fmt.Errorf("secret doesn't have required private key with name %s", corev1.TLSPrivateKeyKey)
	}

	keyBlock, _ := pem.Decode(key)
	if keyBlock == nil {
		return false, fmt.Errorf("failed to parse private key PEM")
	}

	if _, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes); err != nil {
		return false, err
	}

	return true, nil
}

// IsFLBTLSSecret checks if the secret is a valid FLB TLS Secret by checking the label
func IsFLBTLSSecret(secret *corev1.Secret) bool {
	if len(secret.Labels) == 0 {
		return false
	}

	tls, ok := secret.Labels[constants.FLBTLSSecretLabel]
	if !ok {
		return false
	}

	return tls == "true"
}

// GetTLSSecretMode returns the TLS Secret Mode, default is Local
func GetTLSSecretMode(svc *corev1.Service) TLSSecretMode {
	if len(svc.Annotations) == 0 {
		return TLSSecretModeLocal
	}

	mode, ok := svc.Annotations[constants.FLBTLSSecretModeAnnotation]
	if !ok {
		return TLSSecretModeLocal
	}

	mode = strings.ToLower(mode)

	switch TLSSecretMode(mode) {
	case TLSSecretModeRemote, TLSSecretModeLocal:
		return TLSSecretMode(mode)
	}

	return TLSSecretModeLocal
}

// IsTLSEnabled checks if the service is enabled for TLS
func IsTLSEnabled(svc *corev1.Service) bool {
	if len(svc.Annotations) == 0 {
		return false
	}

	return svc.Annotations[constants.FLBTLSEnabledAnnotation] == "true"
}

// IsValidTLSPort checks if the service has valid TLS port
func IsValidTLSPort(svc *corev1.Service) (bool, error) {
	if len(svc.Annotations) == 0 {
		return false, fmt.Errorf("service has empty annotations")
	}

	port, ok := svc.Annotations[constants.FLBTLSPortAnnotation]
	if !ok {
		return false, fmt.Errorf("service doesn't have annotation %s", constants.FLBTLSPortAnnotation)
	}

	p, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return false, err
	}

	if p <= 0 || p > 65535 {
		return false, fmt.Errorf("invalid port number: %d", p)
	}

	// check if the TLS port is in the service spec
	for _, port := range svc.Spec.Ports {
		if port.Port == int32(p) {
			return true, nil
		}
	}

	return false, fmt.Errorf("port %d is not found in service spec", p)
}
