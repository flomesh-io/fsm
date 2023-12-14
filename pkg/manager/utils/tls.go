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

package utils

import (
	"fmt"

	"github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"

	"github.com/tidwall/sjson"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/repo"
)

// UpdateIngressTLSConfig updates TLS config of ingress controller
func UpdateIngressTLSConfig(basepath string, repoClient *repo.PipyRepoClient, mc configurator.Configurator, nsig *v1alpha1.NamespacedIngress) error {
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return err
	}

	newJSON, err := updateTLS(mc, json, nsig)
	if err != nil {
		log.Error().Msgf("Failed to update TLS config: %s", err)
		return err
	}

	return updateMainJSON(basepath, repoClient, newJSON)
}

func updateTLS(mc configurator.Configurator, json string, nsig *v1alpha1.NamespacedIngress) (string, error) {
	var err error

	if nsig != nil && nsig.Spec.TLS != nil {
		enabled := false
		if nsig.Spec.TLS.Enabled != nil {
			enabled = *nsig.Spec.TLS.Enabled
		}

		mTLS := false
		if nsig.Spec.TLS.MTLS != nil {
			mTLS = *nsig.Spec.TLS.MTLS
		}

		for path, value := range map[string]interface{}{
			"tls.enabled": enabled,
			"tls.listen":  nsig.Spec.TLS.Port.Port,
			"tls.mTLS":    mTLS,
		} {
			json, err = sjson.Set(json, path, value)
			if err != nil {
				return "", err
			}
		}

		return json, err
	}

	for path, value := range map[string]interface{}{
		"tls.enabled": mc.IsIngressTLSEnabled(),
		"tls.listen":  mc.GetIngressTLSListenPort(),
		"tls.mTLS":    mc.IsIngressMTLSEnabled(),
	} {
		json, err = sjson.Set(json, path, value)
		if err != nil {
			return "", err
		}
	}

	return json, err
}

// IssueCertForIngress issues certificate for ingress controller
func IssueCertForIngress(basepath string, repoClient *repo.PipyRepoClient, certMgr *certificate.Manager, mc configurator.Configurator, nsig *v1alpha1.NamespacedIngress) error {
	// 1. issue cert
	cert, err := issueCert(certMgr, mc, nsig)
	if err != nil {
		return err
	}

	// 2. get main.json
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return err
	}

	// 3. update tls config
	newJSON, err := updateTLSAndCert(json, mc, cert, nsig)
	if err != nil {
		log.Error().Msgf("Failed to update TLS config: %s", err)
		return err
	}

	// 6. update main.json
	return updateMainJSON(basepath, repoClient, newJSON)
}

func issueCert(certMgr *certificate.Manager, mc configurator.Configurator, nsig *v1alpha1.NamespacedIngress) (*certificate.Certificate, error) {
	cert, err := certMgr.IssueCertificate(
		getCertPrefix(mc, nsig),
		certificate.IngressGateway,
		certificate.FullCNProvided())
	if err != nil {
		log.Error().Msgf("Issue certificate for ingress-pipy error: %s", err)
		return nil, err
	}

	return cert, nil
}

func getCertPrefix(mc configurator.Configurator, nsig *v1alpha1.NamespacedIngress) string {
	if nsig != nil {
		return fmt.Sprintf("%s.%s.svc", nsig.Name, nsig.Namespace)
	}

	return fmt.Sprintf("%s.%s.svc", constants.FSMIngressName, mc.GetFSMNamespace())
}

func updateTLSAndCert(json string, mc configurator.Configurator, cert *certificate.Certificate, nsig *v1alpha1.NamespacedIngress) (string, error) {
	if nsig != nil && nsig.Spec.TLS != nil {
		enabled := false
		if nsig.Spec.TLS.Enabled != nil {
			enabled = *nsig.Spec.TLS.Enabled
		}

		mTLS := false
		if nsig.Spec.TLS.MTLS != nil {
			mTLS = *nsig.Spec.TLS.MTLS
		}

		return sjson.Set(json, "tls", map[string]interface{}{
			"enabled": enabled,
			"listen":  nsig.Spec.TLS.Port.Port,
			"mTLS":    mTLS,
			"certificate": map[string]interface{}{
				"cert": string(cert.GetCertificateChain()),
				"key":  string(cert.GetPrivateKey()),
				"ca":   string(cert.GetIssuingCA()),
			},
		})
	}

	return sjson.Set(json, "tls", map[string]interface{}{
		"enabled": mc.IsIngressTLSEnabled(),
		"listen":  mc.GetIngressTLSListenPort(),
		"mTLS":    mc.IsIngressMTLSEnabled(),
		"certificate": map[string]interface{}{
			"cert": string(cert.GetCertificateChain()),
			"key":  string(cert.GetPrivateKey()),
			"ca":   string(cert.GetIssuingCA()),
		},
	})
}

// UpdateSSLPassthrough updates SSL passthrough config
func UpdateSSLPassthrough(basepath string, repoClient *repo.PipyRepoClient, enabled bool, upstreamPort int32) error {
	log.Info().Msgf("SSL passthrough is enabled, updating repo config ...")
	// 1. get main.json
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return err
	}

	// 2. update ssl passthrough config
	log.Info().Msgf("SSLPassthrough enabled=%t", enabled)
	log.Info().Msgf("SSLPassthrough upstreamPort=%d", upstreamPort)
	newJSON, err := sjson.Set(json, "sslPassthrough", map[string]interface{}{
		"enabled":      enabled,
		"upstreamPort": upstreamPort,
	})
	if err != nil {
		log.Error().Msgf("Failed to update sslPassthrough: %s", err)
		return err
	}

	// 3. update main.json
	return updateMainJSON(basepath, repoClient, newJSON)
}
