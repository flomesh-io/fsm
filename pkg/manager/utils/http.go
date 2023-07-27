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
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/tidwall/sjson"
)

func UpdateIngressHTTPConfig(basepath string, repoClient *client.PipyRepoClient, mc configurator.Configurator) error {
	json, err := getMainJson(basepath, repoClient)
	if err != nil {
		return err
	}

	newJson, err := sjson.Set(json, "http", map[string]interface{}{
		"enabled": mc.IsIngressHTTPEnabled(),
		"listen":  mc.GetIngressHTTPListenPort(),
	})
	if err != nil {
		log.Error().Msgf("Failed to update HTTP config: %s", err)
		return err
	}

	return updateMainJson(basepath, repoClient, newJson)
}
