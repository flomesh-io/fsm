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
	"context"
	"fmt"

	"github.com/tidwall/sjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
)

var (
	loggingEnabledPluginsChain = []string{
		"plugins/reject-http.js",
		"plugins/protocol.js",
		"plugins/router.js",
		"plugins/logging.js",
		"plugins/metrics.js",
		"plugins/balancer.js",
		"plugins/default.js",
	}

	loggingDisabledPluginsChain = []string{
		"plugins/reject-http.js",
		"plugins/protocol.js",
		"plugins/router.js",
		"plugins/balancer.js",
		"plugins/default.js",
	}
)

// UpdateLoggingConfig updates logging config of ingress controller
func UpdateLoggingConfig(kubeClient kubernetes.Interface, basepath string, repoClient *client.PipyRepoClient, mc configurator.Configurator) error {
	json, err := getNewLoggingConfigJSON(kubeClient, basepath, repoClient, mc)
	if err != nil {
		return err
	}

	return updateMainJSON(basepath, repoClient, json)
}

func getNewLoggingConfigJSON(kubeClient kubernetes.Interface, basepath string, repoClient *client.PipyRepoClient, mc configurator.Configurator) (string, error) {
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return "", err
	}

	if mc.IsRemoteLoggingEnabled() {
		secret, err := getLoggingSecret(kubeClient, mc)
		if err != nil {
			return "", err
		}

		if secret == nil {
			return "", fmt.Errorf("secret %q doesn't exist", mc.GetRemoteLoggingSecretName())
		}

		for path, value := range map[string]interface{}{
			"logging.enabled": mc.IsRemoteLoggingEnabled(),
			"logging.url":     string(secret.Data["url"]),
			"logging.token":   string(secret.Data["token"]),
			"plugins":         loggingEnabledPluginsChain,
		} {
			json, err = sjson.Set(json, path, value)
			if err != nil {
				log.Error().Msgf("Failed to update Logging config: %s", err)
				return "", err
			}
		}
	} else {
		for path, value := range map[string]interface{}{
			"logging.enabled": mc.IsRemoteLoggingEnabled(),
			"plugins":         loggingDisabledPluginsChain,
		} {
			json, err = sjson.Set(json, path, value)
			if err != nil {
				log.Error().Msgf("Failed to update Logging config: %s", err)
				return "", err
			}
		}
	}

	return json, nil
}

func getLoggingSecret(kubeClient kubernetes.Interface, mc configurator.Configurator) (*corev1.Secret, error) {
	if mc.IsRemoteLoggingEnabled() {
		secretName := mc.GetRemoteLoggingSecretName()
		secret, err := kubeClient.CoreV1().
			Secrets(mc.GetFSMNamespace()).
			Get(context.TODO(), secretName, metav1.GetOptions{})

		if err != nil {
			if errors.IsNotFound(err) {
				secret, err = kubeClient.CoreV1().
					Secrets(mc.GetFSMNamespace()).
					Create(
						context.TODO(),
						&corev1.Secret{
							TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Name:      secretName,
								Namespace: mc.GetFSMNamespace(),
							},
							Data: map[string][]byte{
								"url":   []byte("http://localhost:8123/ping"),
								"token": []byte("[UNKNOWN]"),
							},
						},
						metav1.CreateOptions{},
					)

				if err != nil {
					log.Error().Msgf("failed to create Secret %s/%s: %s", mc.GetFSMNamespace(), secretName, err)
					return nil, err
				}

				return secret, nil
			}

			log.Error().Msgf("failed to get Secret %s/%s: %s", mc.GetFSMNamespace(), secretName, err)
			return nil, err
		}

		return secret, nil
	}

	return nil, nil
}
