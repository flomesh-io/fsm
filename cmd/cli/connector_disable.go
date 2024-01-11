package main

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
)

const connectorDisableDescription = `
This command will disable FSM connector, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type connectorDisableCmd struct {
	out          io.Writer
	kubeClient   kubernetes.Interface
	configClient configClientset.Interface
	meshName     string
}

func newConnectorDisable(out io.Writer) *cobra.Command {
	disableCmd := &connectorDisableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "disable fsm connector",
		Long:  connectorDisableDescription,
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

			return disableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&disableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")

	return cmd
}

func (cmd *connectorDisableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if !mc.Spec.EgressGateway.Enabled {
		fmt.Fprintf(cmd.out, "egress-gateway is disabled already, no action needed\n")
		return nil
	}

	debug("Deleting FSM egress-gateway resources ...")
	err = deleteEgressGatewayResources(ctx, cmd.kubeClient, fsmNamespace, cmd.meshName)
	if err != nil {
		return err
	}

	err = updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"egressGateway.enabled": false,
	})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.EgressGateway.Enabled = false
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "egress-gateway is disabled successfully\n")

	return nil
}
