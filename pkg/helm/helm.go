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

package helm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/api/meta"

	"k8s.io/client-go/dynamic"

	"helm.sh/helm/v3/pkg/releaseutil"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// RenderChart renders a chart and returns the rendered manifest
func RenderChart(
	templateClient *action.Install,
	object metav1.Object,
	chartSource []byte,
	mc configurator.Configurator,
	client client.Client,
	scheme *runtime.Scheme,
	resolveValues func(metav1.Object, configurator.Configurator) (map[string]interface{}, error),
) (ctrl.Result, error) {
	chart, err := loader.LoadArchive(bytes.NewReader(chartSource))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error loading chart for installation: %s", err)
	}
	log.Debug().Msgf("[HELM UTIL] Chart = %v", chart)

	values, err := resolveValues(object, mc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error resolve values for installation: %s", err)
	}
	log.Debug().Msgf("[HELM UTIL] Values = %s", values)

	rel, err := templateClient.Run(chart, values)
	if err != nil {
		log.Error().Msgf("[HELM UTIL] Error rendering chart: %s", err)
		return ctrl.Result{}, fmt.Errorf("error rendering templates: %s", err)
	}

	manifests := rel.Manifest
	log.Debug().Msgf("[HELM UTIL] Manifest = \n%s\n", manifests)

	if result, err := applyChartYAMLs(object, manifests, client, scheme); err != nil {
		log.Error().Msgf("[HELM UTIL] Error applying chart YAMLs: %s", err)
		return result, err
	}

	return ctrl.Result{}, nil
}

// RenderChartWithValues renders a chart and returns the rendered manifest
func RenderChartWithValues(
	templateClient *action.Install,
	object metav1.Object,
	chartSource []byte,
	client client.Client,
	scheme *runtime.Scheme,
	values map[string]interface{},
) (ctrl.Result, error) {
	return RenderChart(templateClient, object, chartSource, nil, client, scheme, func(metav1.Object, configurator.Configurator) (map[string]interface{}, error) {
		return values, nil
	})
}

func TemplateClient(cfg *action.Configuration, releaseName, namespace string, kubeVersion *chartutil.KubeVersion) *action.Install {
	//log.Debug().Msgf("[HELM UTIL] Creating Helm Install Client ...")
	installClient := action.NewInstall(cfg)
	installClient.ReleaseName = releaseName
	installClient.Namespace = namespace
	installClient.CreateNamespace = false
	installClient.DryRun = true
	installClient.ClientOnly = true
	installClient.KubeVersion = kubeVersion

	return installClient
}

func ActionConfig(namespace string, debugLog action.DebugLog) *action.Configuration {
	configFlags := &genericclioptions.ConfigFlags{Namespace: &namespace}

	actionConfig := new(action.Configuration)
	_ = actionConfig.Init(configFlags, namespace, "secret", debugLog)

	return actionConfig
}

func applyChartYAMLs(owner metav1.Object, manifests string, client client.Client, scheme *runtime.Scheme) (ctrl.Result, error) {
	yamlReader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader([]byte(manifests))))
	for {
		buf, err := yamlReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Error().Msgf("Error reading yaml: %s", err)
			return ctrl.Result{RequeueAfter: 1 * time.Second}, err
		}

		log.Debug().Msgf("[HELM UTIL] Processing YAML : \n\n%s\n\n", string(buf))
		obj, err := utils.DecodeYamlToUnstructured(buf)
		if err != nil {
			log.Error().Msgf("Error decoding YAML to Unstructured object: %s", err)
			return ctrl.Result{RequeueAfter: 1 * time.Second}, err
		}
		log.Debug().Msgf("[HELM UTIL] Unstructured Object = \n\n%v\n\n", obj)

		if isValidOwner(owner, obj) {
			if err = ctrl.SetControllerReference(owner, obj, scheme); err != nil {
				log.Error().Msgf("Error setting controller reference: %s", err)
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}
			log.Debug().Msgf("[HELM UTIL] Resource %s/%s, Owner: %v", obj.GetNamespace(), obj.GetName(), obj.GetOwnerReferences())
		}

		result, err := utils.CreateOrUpdate(context.TODO(), client, obj)
		if err != nil {
			log.Error().Msgf("Error creating/updating object: %s", err)
			return ctrl.Result{RequeueAfter: 1 * time.Second}, err
		}

		log.Debug().Msgf("[HELM UTIL] Successfully %s object: %v", result, obj)
	}

	return ctrl.Result{}, nil
}

func isValidOwner(owner, object metav1.Object) bool {
	ownerNs := owner.GetNamespace()
	if ownerNs != "" {
		objNs := object.GetNamespace()
		if objNs == "" {
			log.Warn().Msgf("cluster-scoped resource must not have a namespace-scoped owner, owner's namespace %s", ownerNs)
			return false
		}
		if ownerNs != objNs {
			log.Warn().Msgf("cross-namespace owner references are disallowed, owner's namespace %s, obj's namespace %s", owner.GetNamespace(), object.GetNamespace())
			return false
		}
	}

	return true
}

func ApplyYAMLs(
	dynamicClient dynamic.Interface,
	mapper meta.RESTMapper,
	manifests string,
	handler YAMLHandlerFunc,
	showFiles ...string,
) error {
	splitManifests := releaseutil.SplitManifests(manifests)
	manifestsKeys := make([]string, 0, len(splitManifests))
	for k := range splitManifests {
		manifestsKeys = append(manifestsKeys, k)
	}
	sort.Sort(releaseutil.BySplitManifestsOrder(manifestsKeys))

	if len(showFiles) > 0 {
		manifestNameRegex := regexp.MustCompile("# Source: [^/]+/(.+)")
		var manifestsToRender []string
		for _, f := range showFiles {
			missing := true
			// Use linux-style filepath separators to unify user's input path
			f = filepath.ToSlash(f)
			for _, manifestKey := range manifestsKeys {
				manifest := splitManifests[manifestKey]
				submatch := manifestNameRegex.FindStringSubmatch(manifest)
				if len(submatch) == 0 {
					continue
				}
				manifestName := submatch[1]
				// manifest.Name is rendered using linux-style filepath separators on Windows as
				// well as macOS/linux.
				manifestPathSplit := strings.Split(manifestName, "/")
				// manifest.Path is connected using linux-style filepath separators on Windows as
				// well as macOS/linux
				manifestPath := strings.Join(manifestPathSplit, "/")

				// if the filepath provided matches a manifest path in the
				// chart, render that manifest
				if matched, _ := filepath.Match(f, manifestPath); !matched {
					continue
				}
				manifestsToRender = append(manifestsToRender, manifest)
				missing = false
			}
			if missing {
				return fmt.Errorf("could not find template %s in chart", f)
			}
		}
		for _, manifest := range manifestsToRender {
			if err := handler(dynamicClient, mapper, manifest); err != nil {
				return err
			}
		}
	} else {
		for _, manifestKey := range manifestsKeys {
			manifest := splitManifests[manifestKey]
			if err := handler(dynamicClient, mapper, manifest); err != nil {
				return err
			}
		}
	}

	return nil
}

func ApplyManifest(dynamicClient dynamic.Interface, mapper meta.RESTMapper, manifest string) error {
	obj, err := utils.DecodeYamlToUnstructured([]byte(manifest))
	if err != nil {
		return err
	}

	if err := utils.CreateOrUpdateUnstructured(context.TODO(), dynamicClient, mapper, obj); err != nil {
		return err
	}

	return nil
}

func DeleteManifest(dynamicClient dynamic.Interface, mapper meta.RESTMapper, manifest string) error {
	obj, err := utils.DecodeYamlToUnstructured([]byte(manifest))
	if err != nil {
		return err
	}

	if err := utils.DeleteUnstructured(context.TODO(), dynamicClient, mapper, obj); err != nil {
		// ignore if not found
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
