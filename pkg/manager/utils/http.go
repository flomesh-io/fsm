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
	"github.com/tidwall/sjson"

	"github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/repo"
)

// UpdateIngressHTTPConfig updates HTTP config of ingress controller
func UpdateIngressHTTPConfig(basepath string, repoClient *repo.PipyRepoClient, mc configurator.Configurator, nsig *v1alpha1.NamespacedIngress) error {
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return err
	}

	newJSON, err := updateHTTPConfig(json, mc, nsig)
	if err != nil {
		log.Error().Msgf("Failed to update HTTP config: %s", err)
		return err
	}

	return updateMainJSON(basepath, repoClient, newJSON)
}

func updateHTTPConfig(json string, mc configurator.Configurator, nsig *v1alpha1.NamespacedIngress) (string, error) {
	var err error

	if nsig != nil {
		for path, value := range map[string]interface{}{
			"http.enabled": nsig.Spec.HTTP.Enabled,
			"http.listen":  nsig.Spec.HTTP.Port.TargetPort,
		} {
			json, err = sjson.Set(json, path, value)
			if err != nil {
				return "", err
			}
		}

		return json, nil
	}

	for path, value := range map[string]interface{}{
		"http.enabled": mc.IsIngressHTTPEnabled(),
		"http.listen":  mc.GetIngressHTTPListenPort(),
	} {
		json, err = sjson.Set(json, path, value)
		if err != nil {
			return "", err
		}
	}

	return json, nil
}
