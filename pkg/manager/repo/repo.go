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

package repo

import (
	"fmt"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/logger"
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/flomesh-io/fsm/pkg/utils"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ScriptsRoot = "/repo/scripts"
)

var (
	log = logger.New("fsm-controller/repo")
)

func InitRepo(ctx *fctx.ControllerContext) error {
	klog.Infof("[MGR] Initializing PIPY Repo ...")
	// wait until pipy repo is up or timeout after 5 minutes
	repoClient := ctx.RepoClient

	if err := wait.PollImmediate(5*time.Second, 60*5*time.Second, func() (bool, error) {
		success, err := repoClient.IsRepoUp()
		if success {
			log.Info().Msg("Repo is READY!")
			return success, nil
		}
		log.Error().Msg("Repo is not up, sleeping ...")
		return success, err
	}); err != nil {
		klog.Errorf("Error happened while waiting for repo up, %s", err)
		return err
	}

	mc := ctx.Config
	// initialize the repo
	if _, err := repoClient.Batch(fmt.Sprintf("%d", 0), getBatches(mc)); err != nil {
		return err
	}

	// derive codebase
	// Services
	defaultServicesPath := utils.GetDefaultServicesPath()
	if _, err := repoClient.DeriveCodebase(defaultServicesPath, constants.DefaultServiceBasePath, 0); err != nil {
		return err
	}

	// Ingress
	if mc.IsIngressEnabled() {
		defaultIngressPath := utils.GetDefaultIngressPath()
		if _, err := repoClient.DeriveCodebase(defaultIngressPath, constants.DefaultIngressBasePath, 0); err != nil {
			return err
		}
	}

	// GatewayAPI
	if mc.IsGatewayApiEnabled() {
		defaultGatewaysPath := utils.GetDefaultGatewaysPath()
		if _, err := repoClient.DeriveCodebase(defaultGatewaysPath, constants.DefaultGatewayBasePath, 0); err != nil {
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

	if mc.IsGatewayApiEnabled() {
		batches = append(batches, gatewaysBatch())
	}

	return batches
}

func ingressBatch() repo.Batch {
	return createBatch(constants.DefaultIngressBasePath, fmt.Sprintf("%s/ingress", ScriptsRoot))
}

func servicesBatch() repo.Batch {
	return createBatch(constants.DefaultServiceBasePath, fmt.Sprintf("%s/services", ScriptsRoot))
}

func gatewaysBatch() repo.Batch {
	return createBatch(constants.DefaultGatewayBasePath, fmt.Sprintf("%s/gateways", ScriptsRoot))
}

func createBatch(repoPath, scriptsDir string) repo.Batch {
	batch := repo.Batch{
		Basepath: repoPath,
		Items:    []repo.BatchItem{},
	}

	for _, file := range listFiles(scriptsDir) {
		content, err := ioutil.ReadFile(file)
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
			klog.Errorf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() {
			*files = append(*files, path)
		}

		return nil
	}
}
