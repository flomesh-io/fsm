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
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/tidwall/sjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
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

func UpdateLoggingConfig(kubeClient kubernetes.Interface, basepath string, repoClient *client.PipyRepoClient, mc configurator.Configurator) error {
	json, err := getNewLoggingConfigJson(kubeClient, basepath, repoClient, mc)
	if err != nil {
		return err
	}

	return updateMainJson(basepath, repoClient, json)
}

func getNewLoggingConfigJson(kubeClient kubernetes.Interface, basepath string, repoClient *client.PipyRepoClient, mc configurator.Configurator) (string, error) {
	json, err := getMainJson(basepath, repoClient)
	if err != nil {
		return "", err
	}

	if mc.Logging.Enabled {
		secret, err := getLoggingSecret(kubeClient, mc)
		if err != nil {
			return "", err
		}

		if secret == nil {
			return "", fmt.Errorf("secret %q doesn't exist", mc.Logging.SecretName)
		}

		for path, value := range map[string]interface{}{
			"logging.enabled": mc.Logging.Enabled,
			"logging.url":     string(secret.Data["url"]),
			"logging.token":   string(secret.Data["token"]),
			"plugins":         loggingEnabledPluginsChain,
		} {
			json, err = sjson.Set(json, path, value)
			if err != nil {
				klog.Errorf("Failed to update Logging config: %s", err)
				return "", err
			}
		}
	} else {
		for path, value := range map[string]interface{}{
			"logging.enabled": mc.Logging.Enabled,
			"plugins":         loggingDisabledPluginsChain,
		} {
			json, err = sjson.Set(json, path, value)
			if err != nil {
				klog.Errorf("Failed to update Logging config: %s", err)
				return "", err
			}
		}
	}

	return json, nil
}

func getLoggingSecret(kubeClient kubernetes.Interface, mc configurator.Configurator) (*corev1.Secret, error) {
	if mc.Logging.Enabled {
		secretName := mc.Logging.SecretName
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
					klog.Errorf("failed to create Secret %s/%s: %s", mc.GetFSMNamespace(), secretName, err)
					return nil, err
				}

				return secret, nil
			}

			klog.Errorf("failed to get Secret %s/%s: %s", mc.GetFSMNamespace(), secretName, err)
			return nil, err
		}

		return secret, nil
	}

	return nil, nil
}
