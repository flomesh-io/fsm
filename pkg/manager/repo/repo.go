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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	fctx "github.com/flomesh-io/fsm/pkg/context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	mutils "github.com/flomesh-io/fsm/pkg/manager/utils"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
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
func InitRepo(ctx context.Context) error {
	log.Info().Msgf("[MGR] Initializing PIPY Repo ...")

	cctx, err := fctx.ToControllerContext(ctx)
	if err != nil {
		return err
	}

	// wait until pipy repo is up or timeout after 5 minutes
	repoClient := cctx.RepoClient

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

	mc := cctx.Configurator
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

//// ChecksAndRebuildRepo checks and rebuilds the repo periodically if needed
//func ChecksAndRebuildRepo(repoClient *repo.PipyRepoClient, client client.Client, mc configurator.Configurator) {
//	s := gocron.NewScheduler(time.Local)
//	s.SingletonModeAll()
//	if _, err := s.Every(60).Seconds().
//		Name("rebuild-repo").
//		Do(rebuildRepoJob, repoClient, client, mc); err != nil {
//		log.Error().Msgf("Error happened while rebuilding repo: %s", err)
//	}
//	s.RegisterEventListeners(
//		gocron.AfterJobRuns(func(jobName string) {
//			log.Debug().Msgf(">>>>>> After ChecksAndRebuildRepo: %s\n", jobName)
//		}),
//		gocron.BeforeJobRuns(func(jobName string) {
//			log.Debug().Msgf(">>>>>> Before ChecksAndRebuildRepo: %s\n", jobName)
//		}),
//		gocron.WhenJobReturnsError(func(jobName string, err error) {
//			log.Error().Msgf(">>>>>> ChecksAndRebuildRepo Returns Error: %s, %v\n", jobName, err)
//		}),
//	)
//	s.StartAsync()
//}

func (r *rebuilder) rebuildRepoJob() error {
	log.Debug().Msg("<<<<<< rebuilding repo - start >>>>>> ")

	if !r.repoClient.IsRepoUp() {
		log.Debug().Msg("Repo is not up, sleeping ...")
		return nil
	}

	// initialize the repo
	batches := make([]repo.Batch, 0)
	if !r.repoClient.CodebaseExists(constants.DefaultIngressBasePath) {
		batches = append(batches, ingressBatch())
	}
	if !r.repoClient.CodebaseExists(constants.DefaultServiceBasePath) {
		batches = append(batches, servicesBatch())
	}
	if !r.repoClient.CodebaseExists(constants.DefaultGatewayBasePath) {
		batches = append(batches, gatewaysBatch())
	}

	if len(batches) > 0 {
		if err := r.repoClient.Batch(batches); err != nil {
			log.Error().Msgf("Failed to write config to repo: %s", err)
			return err
		}
	}

	defaultServicesPath := utils.GetDefaultServicesPath()
	if err := r.repoClient.DeriveCodebase(defaultServicesPath, constants.DefaultServiceBasePath); err != nil {
		log.Error().Msgf("%q failed to derive codebase %q: %s", defaultServicesPath, constants.DefaultServiceBasePath, err)
		return err
	}

	if r.mc.IsIngressEnabled() {
		defaultIngressPath := utils.GetDefaultIngressPath()
		if err := r.repoClient.DeriveCodebase(defaultIngressPath, constants.DefaultIngressBasePath); err != nil {
			log.Error().Msgf("%q failed to derive codebase %q: %s", defaultIngressPath, constants.DefaultIngressBasePath, err)
			return err
		}

		if err := mutils.UpdateMainVersion(constants.DefaultIngressBasePath, r.repoClient, r.mc); err != nil {
			log.Error().Msgf("Failed to update version of main.json: %s", err)
			return err
		}
	}

	if r.mc.IsNamespacedIngressEnabled() {
		nsigList := &nsigv1alpha1.NamespacedIngressList{}
		if err := r.client.List(context.TODO(), nsigList, client.InNamespace(corev1.NamespaceAll)); err != nil {
			return err
		}

		for _, nsig := range nsigList.Items {
			ingressPath := utils.NamespacedIngressCodebasePath(nsig.Namespace)
			parentPath := utils.IngressCodebasePath()
			if err := r.repoClient.DeriveCodebase(ingressPath, parentPath); err != nil {
				log.Error().Msgf("Codebase of NamespaceIngress %q failed to derive codebase %q: %s", ingressPath, parentPath, err)
				return err
			}
		}
	}

	if r.mc.IsGatewayAPIEnabled() {
		defaultGatewaysPath := utils.GetDefaultGatewaysPath()
		if err := r.repoClient.DeriveCodebase(defaultGatewaysPath, constants.DefaultGatewayBasePath); err != nil {
			log.Error().Msgf("%q failed to derive codebase %q: %s", defaultGatewaysPath, constants.DefaultGatewayBasePath, err)
			return err
		}

		gatewayList := &gwv1.GatewayList{}
		if err := r.client.List(
			context.TODO(),
			gatewayList,
			client.InNamespace(corev1.NamespaceAll),
		); err != nil {
			log.Error().Msgf("Failed to list all gateways: %s", err)
			return err
		}

		log.Debug().Msgf("Found %d gateways", len(gatewayList.Items))

		for _, gw := range gatewayList.Items {
			gw := gw // fix lint GO-LOOP-REF
			if gwutils.IsActiveGateway(&gw) {
				gwPath := utils.GatewayCodebasePath(gw.Namespace)
				parentPath := utils.GetDefaultGatewaysPath()
				if err := r.repoClient.DeriveCodebase(gwPath, parentPath); err != nil {
					return err
				}
			}
		}
	}

	log.Debug().Msg("<<<<<< rebuilding repo - end >>>>>> ")
	return nil
}
