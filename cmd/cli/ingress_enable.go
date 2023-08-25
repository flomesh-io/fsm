package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"helm.sh/helm/v3/pkg/action"

	metautil "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/restmapper"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"helm.sh/helm/v3/pkg/chart/loader"

	"k8s.io/client-go/dynamic"

	"github.com/flomesh-io/fsm/pkg/helm"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/tidwall/sjson"

	"k8s.io/apimachinery/pkg/types"

	"github.com/flomesh-io/fsm/pkg/constants"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const ingressEnableDescription = `
This command will enable metrics scraping on all pods belonging to the given
namespace or set of namespaces. Newly created pods belonging to namespaces that
are enabled for metrics will be automatically enabled with metrics.

The command does not deploy a metrics collection service such as Prometheus.
`

const (
	presetMeshConfigName    = "preset-mesh-config"
	presetMeshConfigJSONKey = "preset-mesh-config.json"
)

var (
	ingressManifestFiles = []string{
		"templates/fsm-ingress-class.yaml",
		"templates/fsm-ingress-deployment.yaml",
		"templates/fsm-ingress-service.yaml",
	}
)

type ingressEnableCmd struct {
	out           io.Writer
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
	configClient  configClientset.Interface
	meshName      string
	mapper        metautil.RESTMapper
	actionConfig  *action.Configuration
}

func newIngressEnable(config *action.Configuration, out io.Writer) *cobra.Command {
	enableCmd := &ingressEnableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "enable ...",
		Short: "enable ingress",
		Long:  ingressEnableDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			enableCmd.actionConfig = config

			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("error fetching kubeconfig: %w", err)
			}

			kubeClient, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			enableCmd.kubeClient = kubeClient

			dynamicClient, err := dynamic.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			enableCmd.dynamicClient = dynamicClient

			gr, err := restmapper.GetAPIGroupResources(kubeClient.Discovery())
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}

			mapper := restmapper.NewDiscoveryRESTMapper(gr)
			enableCmd.mapper = mapper

			configClient, err := configClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			enableCmd.configClient = configClient

			return enableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&enableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")

	return cmd
}

func (cmd *ingressEnableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Checking fsm release in namespace %q", fsmNamespace)
	actionConfig := helm.ActionConfig(fsmNamespace, debug)

	listClient := action.NewList(actionConfig)
	releases, err := listClient.Run()
	if err != nil {
		return err
	}

	switch len(releases) {
	case 0:
		fmt.Fprintf(cmd.out, "No existing fsm release in namespace %q \n", fsmNamespace)
		return nil
	case 1:
		fmt.Fprintf(cmd.out, "Found existing fsm release %q in namespace %q \n", releases[0].Name, fsmNamespace)
	default:
		fmt.Fprintf(cmd.out, "Found %d existing fsm releases in namespace %q \n", len(releases), fsmNamespace)
		return nil
	}

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// check if ingress is enabled, if yes, just return
	if mc.Spec.Ingress.Enabled {
		fmt.Fprintf(cmd.out, "Ingress is enabled, not action needed")
		return nil
	}

	debug("Getting configmap preset-mesh-config ...")
	// get configmap preset-mesh-config
	cm, err := cmd.kubeClient.CoreV1().ConfigMaps(fsmNamespace).Get(ctx, presetMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	debug("Updating configmap preset-mesh-config ...")
	// update content data of preset-mesh-config.json
	json := cm.Data[presetMeshConfigJSONKey]
	newJson, err := sjson.Set(json, "ingress", map[string]interface{}{"enabled": true, "namespaced": false})
	if err != nil {
		return err
	}
	newJson, err = sjson.Set(newJson, "gatewayAPI", map[string]interface{}{"enabled": false})
	if err != nil {
		return err
	}

	// update configmap preset-mesh-config
	cm.Data[presetMeshConfigJSONKey] = newJson
	_, err = cmd.kubeClient.CoreV1().ConfigMaps(fsmNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.Ingress.Enabled = true
	mc.Spec.Ingress.Namespaced = false
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	debug("Restarting fsm-controller ...")
	// Rollout restart fsm-controller
	// patch the deployment spec template triggers the action of rollout restart like with kubectl
	patch := fmt.Sprintf(
		`{"spec": {"template":{"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`,
		time.Now().Format("20060102-150405.0000"),
	)

	_, err = cmd.kubeClient.AppsV1().
		Deployments(fsmNamespace).
		Patch(ctx, constants.FSMControllerName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return err
	}

	debug("Loading fsm helm chart ...")
	// load fsm helm chart
	chart, err := loader.LoadArchive(bytes.NewReader(chartTGZSource))
	if err != nil {
		return err
	}

	debug("Resolving values ...")
	// resolve values
	values, err := cmd.resolveValues(mc)
	if err != nil {
		return err
	}

	debug("Creating helm template client ...")
	// create a helm template client
	templateClient := helm.TemplateClient(
		actionConfig,
		releases[0].Name,
		fsmNamespace,
		&chartutil.KubeVersion{
			Version: fmt.Sprintf("v%s.%s.0", "1", "19"),
			Major:   "1",
			Minor:   "19",
		},
	)
	templateClient.Replace = true

	debug("Rendering helm template ...")
	// render entire fsm helm template
	rel, err := templateClient.Run(chart, values)
	if err != nil {
		return err
	}
	//debug("rel.Config = %s", rel.Config)
	//debug("rel.Manifest = %s", rel.Manifest)

	debug("Apply ingress manifests ...")
	// filter out unneeded manifests, only keep ingress manifests, then do a kubectl-apply like action for each manifest
	if err := helm.ApplyYAMLs(cmd.dynamicClient, cmd.mapper, rel.Manifest, ingressManifestFiles...); err != nil {
		return err
	}

	// TODO: wait for pod ready? no hurry

	return nil
}

func (cmd *ingressEnableCmd) resolveValues(mc *configv1alpha3.MeshConfig) (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	valuesConfig := []string{
		fmt.Sprintf("fsm.fsmIngress.enabled=%t", true),
		fmt.Sprintf("fsm.fsmIngress.namespaced=%t", false),
		fmt.Sprintf("fsm.fsmGateway.enabled=%t", false),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetNamespace()),
		fmt.Sprintf("fsm.meshName=%s", cmd.meshName),
		fmt.Sprintf("fsm.image.registry=%s", mc.Spec.Image.Registry),
		fmt.Sprintf("fsm.image.pullPolicy=%s", mc.Spec.Image.PullPolicy),
		fmt.Sprintf("fsm.image.tag=%s", mc.Spec.Image.Tag),
		fmt.Sprintf("fsm.curlImage=%s", mc.Spec.Misc.CurlImage),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return nil, err
	}

	return finalValues, nil
}
