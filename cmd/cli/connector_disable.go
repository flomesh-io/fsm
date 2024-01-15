package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
)

const connectorDisableDescription = `
This command will disable FSM connector, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type connectorDisableCmd struct {
	out           io.Writer
	kubeClient    kubernetes.Interface
	configClient  configClientset.Interface
	meshName      string
	connectorName string
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
	f.StringVar(&disableCmd.connectorName, "connector-name", "", "name for the fsm connector instance")

	return cmd
}

func (cmd *connectorDisableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(cmd.connectorName) == 0 {
		return errors.New("missing connector name")
	}

	fsmNamespace := settings.Namespace()

	debug("Deleting FSM connector resources ...")
	err := deleteConnectorResources(ctx, cmd.kubeClient, fsmNamespace, cmd.meshName, cmd.connectorName)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "%s is disabled successfully\n", cmd.connectorName)

	return nil
}
