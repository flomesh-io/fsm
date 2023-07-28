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
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/flomesh-io/fsm/pkg/utils"
)

func getMainJson(basepath string, repoClient *repo.PipyRepoClient) (string, error) {
	path := getPathOfMainJson(basepath)

	json, err := repoClient.GetFile(path)
	if err != nil {
		log.Error().Msgf("Get %q from pipy repo error: %s", path, err)
		return "", err
	}

	return json, nil
}

func updateMainJson(basepath string, repoClient *repo.PipyRepoClient, newJson string) error {
	batch := repo.Batch{
		Basepath: basepath,
		Items: []repo.BatchItem{
			{
				Path:     "/config",
				Filename: "main.json",
				Content:  newJson,
			},
		},
	}

	hash := utils.Hash([]byte(newJson))
	if _, err := repoClient.Batch(fmt.Sprintf("%d", hash), []repo.Batch{batch}); err != nil {
		log.Error().Msgf("Failed to update %q: %s", getPathOfMainJson(basepath), err)
		return err
	}

	return nil
}

func getPathOfMainJson(basepath string) string {
	return fmt.Sprintf("%s/config/main.json", basepath)
}
