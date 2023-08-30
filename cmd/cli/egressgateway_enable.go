package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"k8s.io/utils/pointer"

	"github.com/flomesh-io/fsm/pkg/constants"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/helm"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

const egressGatewayEnableDescription = `
This command will enable FSM egress-gateway, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type egressGatewayEnableCmd struct {
	out           io.Writer
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
	configClient  configClientset.Interface
	mapper        meta.RESTMapper
	actionConfig  *action.Configuration
	meshName      string
	mode          string
	logLevel      string
	adminPort     int32
	port          int32
	replicas      int32
}

func newEgressGatewayEnable(actionConfig *action.Configuration, out io.Writer) *cobra.Command {
	enableCmd := &egressGatewayEnableCmd{
		out:          out,
		actionConfig: actionConfig,
	}

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable fsm egress-gateway",
		Long:  egressGatewayEnableDescription,
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

			return enableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&enableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")
	f.StringVar(&enableCmd.mode, "mode", constants.EgressGatewayModeHTTP2Tunnel, "mode of the egress-gateway, http2tunnel or sock5")
	f.StringVar(&enableCmd.logLevel, "log-level", "error", "log level of egress-gateway")
	f.Int32Var(&enableCmd.adminPort, "admin-port", 6060, "admin port of egress-gateway, rarely need to be set manually")
	f.Int32Var(&enableCmd.port, "port", 1080, "serving port of egress-gateway")
	f.Int32Var(&enableCmd.replicas, "replicas", 1, "replicas of egress-gateway")
	//utilruntime.Must(cmd.MarkFlagRequired("mode"))

	return cmd
}

func (cmd *egressGatewayEnableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cmd.mode != constants.EgressGatewayModeHTTP2Tunnel && cmd.mode != constants.EgressGatewayModeSock5 {
		return fmt.Errorf("mode must be either http2tunnel or socks5")
	}

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if mc.Spec.EgressGateway.Enabled {
		fmt.Fprintf(cmd.out, "egress-gateway is enabled, not action needed")
		return nil
	}

	err = updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"egressGateway": map[string]interface{}{
			"enabled":   true,
			"logLevel":  cmd.logLevel,
			"mode":      cmd.mode,
			"port":      cmd.port,
			"adminPort": cmd.adminPort,
			"replicas":  cmd.replicas,
		},
	})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.EgressGateway = configv1alpha3.EgressGatewaySpec{
		Enabled:   true,
		LogLevel:  cmd.logLevel,
		Mode:      cmd.mode,
		Port:      pointer.Int32(cmd.port),
		AdminPort: pointer.Int32(cmd.adminPort),
		Replicas:  pointer.Int32(cmd.replicas),
	}
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
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

	debug("Apply ingress manifests ...")
	// filter out unneeded manifests, only keep ingress manifests, then do a kubectl-apply like action for each manifest
	if err := helm.ApplyYAMLs(cmd.dynamicClient, cmd.mapper, rel.Manifest, helm.ApplyManifest, egressGatewayManifestFiles...); err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "egress-gateway is enabled successfully\n")

	return nil
}

func (cmd *egressGatewayEnableCmd) resolveValues(mc *configv1alpha3.MeshConfig) (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	valuesConfig := []string{
		fmt.Sprintf("fsm.egressGateway.enabled=%t", true),
		fmt.Sprintf("fsm.egressGateway.logLevel=%s", cmd.logLevel),
		fmt.Sprintf("fsm.egressGateway.mode=%s", cmd.mode),
		fmt.Sprintf("fsm.egressGateway.port=%d", cmd.port),
		fmt.Sprintf("fsm.egressGateway.adminPort=%d", cmd.adminPort),
		fmt.Sprintf("fsm.egressGateway.replicas=%d", cmd.replicas),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetNamespace()),
		fmt.Sprintf("fsm.meshName=%s", cmd.meshName),
		fmt.Sprintf("fsm.image.registry=%s", mc.Spec.Image.Registry),
		fmt.Sprintf("fsm.image.pullPolicy=%s", mc.Spec.Image.PullPolicy),
		fmt.Sprintf("fsm.image.tag=%s", mc.Spec.Image.Tag),
		fmt.Sprintf("fsm.repoServer.image=%s", mc.Spec.Misc.RepoServerImage),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return nil, err
	}

	return finalValues, nil
}
