package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/tidwall/sjson"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/helm"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

const ingressDisableDescription = `
This command will enable metrics scraping on all pods belonging to the given
namespace or set of namespaces. Newly created pods belonging to namespaces that
are enabled for metrics will be automatically enabled with metrics.

The command does not deploy a metrics collection service such as Prometheus.
`

type ingressDisableCmd struct {
	out            io.Writer
	kubeClient     kubernetes.Interface
	dynamicClient  dynamic.Interface
	configClient   configClientset.Interface
	meshName       string
	mapper         meta.RESTMapper
	actionConfig   *action.Configuration
	templateClient *action.Install
}

func newIngressDisable(actionConfig *action.Configuration, out io.Writer) *cobra.Command {
	disableCmd := &ingressDisableCmd{
		out:          out,
		actionConfig: actionConfig,
	}

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "disable ingress",
		Long:  ingressDisableDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("error fetching kubeconfig: %w", err)
			}

			kubeClient, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			disableCmd.kubeClient = kubeClient

			dynamicClient, err := dynamic.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			disableCmd.dynamicClient = dynamicClient

			gr, err := restmapper.GetAPIGroupResources(kubeClient.Discovery())
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}

			mapper := restmapper.NewDiscoveryRESTMapper(gr)
			disableCmd.mapper = mapper

			configClient, err := configClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			disableCmd.configClient = configClient

			return disableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&disableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")
	//utilruntime.Must(cmd.MarkFlagRequired("mesh-name"))

	return cmd
}

func (cmd *ingressDisableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// check if ingress is enabled, if yes, just return
	if !mc.Spec.Ingress.Enabled {
		fmt.Fprintf(cmd.out, "Ingress is disabled, not action needed")
		return nil
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
		cmd.actionConfig,
		cmd.meshName,
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

	debug("Deleting ingress manifests ...")
	if err := helm.ApplyYAMLs(cmd.dynamicClient, cmd.mapper, rel.Manifest, helm.DeleteManifest, ingressManifestFiles...); err != nil {
		return err
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
	for path, value := range map[string]interface{}{
		"ingress.enabled":    false,
		"ingress.namespaced": false,
	} {
		json, err = sjson.Set(json, path, value)
		if err != nil {
			return err
		}
	}

	// update configmap preset-mesh-config
	cm.Data[presetMeshConfigJSONKey] = json
	_, err = cmd.kubeClient.CoreV1().ConfigMaps(fsmNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.Ingress.Enabled = false
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

	fmt.Fprintf(cmd.out, "Ingress is disabled successfully\n")

	return nil
}

func (cmd *ingressDisableCmd) resolveValues(mc *configv1alpha3.MeshConfig) (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	valuesConfig := []string{
		fmt.Sprintf("fsm.fsmIngress.enabled=%t", true), // must be true, otherwise the ingress manifests will not be rendered
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
