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

	"github.com/tidwall/sjson"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
)

// UpdateIngressTLSConfig updates TLS config of ingress controller
func UpdateIngressTLSConfig(basepath string, repoClient *repo.PipyRepoClient, mc configurator.Configurator) error {
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return err
	}

	for path, value := range map[string]interface{}{
		"tls.enabled": mc.IsIngressTLSEnabled(),
		"tls.listen":  mc.GetIngressTLSListenPort(),
		"tls.mTLS":    mc.IsIngressMTLSEnabled(),
	} {
		json, err = sjson.Set(json, path, value)
		if err != nil {
			log.Error().Msgf("Failed to update TLS config: %s", err)
			return err
		}
	}

	return updateMainJSON(basepath, repoClient, json)
}

// IssueCertForIngress issues certificate for ingress controller
func IssueCertForIngress(basepath string, repoClient *client.PipyRepoClient, certMgr *certificate.Manager, mc configurator.Configurator) error {
	// 1. issue cert
	cert, err := certMgr.IssueCertificate(
		fmt.Sprintf("%s.%s.svc", constants.FSMIngressName, mc.GetFSMNamespace()),
		certificate.Internal,
		certificate.FullCNProvided())
	if err != nil {
		log.Error().Msgf("Issue certificate for ingress-pipy error: %s", err)
		return err
	}

	// 2. get main.json
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return err
	}

	newJSON, err := sjson.Set(json, "tls", map[string]interface{}{
		"enabled": mc.IsIngressTLSEnabled(),
		"listen":  mc.GetIngressTLSListenPort(),
		"mTLS":    mc.IsIngressMTLSEnabled(),
		"certificate": map[string]interface{}{
			"cert": cert.GetCertificateChain(),
			"key":  cert.GetPrivateKey(),
			"ca":   cert.GetIssuingCA(),
		},
	})
	if err != nil {
		log.Error().Msgf("Failed to update TLS config: %s", err)
		return err
	}

	// 6. update main.json
	return updateMainJSON(basepath, repoClient, newJSON)
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
