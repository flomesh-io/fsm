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

// Package repo contains the repo utilities
package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

const (
	scriptsRoot = "/repo/scripts"
)

var (
	log = logger.New("fsm-controller/repo")
)

// InitRepo initializes the pipy repo
func InitRepo(ctx *fctx.ControllerContext) error {
	log.Info().Msgf("[MGR] Initializing PIPY Repo ...")
	// wait until pipy repo is up or timeout after 5 minutes
	repoClient := ctx.RepoClient

	if err := wait.PollImmediate(5*time.Second, 60*5*time.Second, func() (bool, error) {
		if repoClient.IsRepoUp() {
			log.Info().Msgf("Repo is READY!")
			return true, nil
		}

		log.Info().Msgf("Repo is not up, sleeping ...")
		return false, nil
	}); err != nil {
		log.Error().Msgf("Error happened while waiting for repo up, %s", err)
		return err
	}

	mc := ctx.Config
	// initialize the repo
	if err := repoClient.Batch(getBatches(mc)); err != nil {
		return err
	}

	// derive codebase
	// Services
	defaultServicesPath := utils.GetDefaultServicesPath()
	if err := repoClient.DeriveCodebase(defaultServicesPath, constants.DefaultServiceBasePath); err != nil {
		return err
	}

	// Ingress
	if mc.IsIngressEnabled() {
		defaultIngressPath := utils.GetDefaultIngressPath()
		if err := repoClient.DeriveCodebase(defaultIngressPath, constants.DefaultIngressBasePath); err != nil {
			return err
		}
	}

	// GatewayAPI
	if mc.IsGatewayAPIEnabled() {
		defaultGatewaysPath := utils.GetDefaultGatewaysPath()
		if err := repoClient.DeriveCodebase(defaultGatewaysPath, constants.DefaultGatewayBasePath); err != nil {
			return err
		}
	}

	return nil
}

func getBatches(mc configurator.Configurator) []repo.Batch {
	batches := []repo.Batch{servicesBatch()}

	if mc.IsIngressEnabled() {
		batches = append(batches, ingressBatch())
	}

	if mc.IsGatewayAPIEnabled() {
		batches = append(batches, gatewaysBatch())
	}

	return batches
}

func ingressBatch() repo.Batch {
	return createBatch(constants.DefaultIngressBasePath, fmt.Sprintf("%s/ingress", scriptsRoot))
}

func servicesBatch() repo.Batch {
	return createBatch(constants.DefaultServiceBasePath, fmt.Sprintf("%s/services", scriptsRoot))
}

func gatewaysBatch() repo.Batch {
	return createBatch(constants.DefaultGatewayBasePath, fmt.Sprintf("%s/gateways", scriptsRoot))
}

func createBatch(repoPath, scriptsDir string) repo.Batch {
	batch := repo.Batch{
		Basepath: repoPath,
		Items:    []repo.BatchItem{},
	}

	for _, file := range listFiles(scriptsDir) {
		content, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			panic(err)
		}

		balancerItem := repo.BatchItem{
			Path:     strings.TrimPrefix(filepath.Dir(file), scriptsDir),
			Filename: filepath.Base(file),
			Content:  string(content),
		}
		batch.Items = append(batch.Items, balancerItem)
	}

	return batch
}

func listFiles(root string) (files []string) {
	err := filepath.Walk(root, visit(&files))

	if err != nil {
		panic(err)
	}

	return files
}

func visit(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Msgf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() {
			*files = append(*files, path)
		}

		return nil
	}
}
