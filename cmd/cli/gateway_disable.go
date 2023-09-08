package main

import (
	"context"
	"fmt"
	"io"

	"github.com/flomesh-io/fsm/pkg/version"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
)

const gatewayDisableDescription = `
This command will disable FSM gateway, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type gatewayDisableCmd struct {
	out              io.Writer
	kubeClient       kubernetes.Interface
	configClient     configClientset.Interface
	gatewayAPIClient gatewayApiClientset.Interface
	meshName         string
}

func newGatewayDisable(out io.Writer) *cobra.Command {
	disableCmd := &gatewayDisableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "disable fsm gateway",
		Long:  gatewayDisableDescription,
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

			configClient, err := configClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			disableCmd.configClient = configClient

			gatewayAPIClient, err := gatewayApiClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			disableCmd.gatewayAPIClient = gatewayAPIClient

			return disableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&disableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")
	//utilruntime.Must(cmd.MarkFlagRequired("mesh-name"))

	return cmd
}

func (cmd *gatewayDisableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !version.IsSupportedK8sVersionForGatewayAPI(cmd.kubeClient) {
		return fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersionForGatewayAPI.String())
	}

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if !mc.Spec.GatewayAPI.Enabled {
		fmt.Fprintf(cmd.out, "Gateway is disabled already, no action needed\n")
		return nil
	}

	debug("Deleting FSM Gateway resources ...")
	err = deleteGatewayResources(ctx, cmd.gatewayAPIClient)
	if err != nil {
		return err
	}

	err = updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"gatewayAPI.enabled": false,
	})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.GatewayAPI.Enabled = false
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if err := restartFSMController(ctx, cmd.kubeClient, fsmNamespace, cmd.out); err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "Gateway is disabled successfully\n")

	return nil
}
