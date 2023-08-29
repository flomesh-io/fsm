package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"

	"helm.sh/helm/v3/pkg/action"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/restmapper"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"helm.sh/helm/v3/pkg/chart/loader"

	"k8s.io/client-go/dynamic"

	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/flomesh-io/fsm/pkg/helm"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const ingressEnableDescription = `
This command will enable FSM ingress, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type ingressEnableCmd struct {
	out              io.Writer
	kubeClient       kubernetes.Interface
	dynamicClient    dynamic.Interface
	configClient     configClientset.Interface
	nsigClient       nsigClientset.Interface
	gatewayAPIClient gatewayApiClientset.Interface
	meshName         string
	mapper           meta.RESTMapper
	actionConfig     *action.Configuration
}

func newIngressEnable(actionConfig *action.Configuration, out io.Writer) *cobra.Command {
	enableCmd := &ingressEnableCmd{
		out:          out,
		actionConfig: actionConfig,
	}

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable fsm ingress",
		Long:  ingressEnableDescription,
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

			nsigClient, err := nsigClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			enableCmd.nsigClient = nsigClient

			gatewayAPIClient, err := gatewayApiClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			enableCmd.gatewayAPIClient = gatewayAPIClient

			return enableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&enableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")
	//utilruntime.Must(cmd.MarkFlagRequired("mesh-name"))

	return cmd
}

func (cmd *ingressEnableCmd) run() error {
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
	// TODO: check if ingress controller is installed and running???
	if mc.Spec.Ingress.Enabled {
		fmt.Fprintf(cmd.out, "Ingress is enabled, not action needed")
		return nil
	}

	debug("Deleting FSM NamespacedIngress resources ...")
	err = deleteNamespacedIngressResources(ctx, cmd.nsigClient)
	if err != nil {
		return err
	}

	debug("Deleting FSM Gateway resources ...")
	err = deleteGatewayResources(ctx, cmd.gatewayAPIClient)
	if err != nil {
		return err
	}

	err = updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"ingress.enabled":    true,
		"ingress.namespaced": false,
		"gatewayAPI.enabled": false,
	})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.Ingress.Enabled = true
	mc.Spec.Ingress.Namespaced = false
	mc.Spec.GatewayAPI.Enabled = false
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	debug("Restarting fsm-controller ...")
	// Rollout restart fsm-controller
	// patch the deployment spec template triggers the action of rollout restart like with kubectl
	err = restartFSMControllerContainer(ctx, cmd.kubeClient, fsmNamespace)
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
	//debug("rel.Config = %s", rel.Config)
	//debug("rel.Manifest = %s", rel.Manifest)

	debug("Apply ingress manifests ...")
	// filter out unneeded manifests, only keep ingress manifests, then do a kubectl-apply like action for each manifest
	if err := helm.ApplyYAMLs(cmd.dynamicClient, cmd.mapper, rel.Manifest, helm.ApplyManifest, ingressManifestFiles...); err != nil {
		return err
	}

	// TODO: wait for pod ready? no hurry

	fmt.Fprintf(cmd.out, "Ingress is enabled successfully\n")

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
