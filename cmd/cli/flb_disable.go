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

const flbDisableDescription = `
This command will disable FSM FLB, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type flbDisableCmd struct {
	out          io.Writer
	kubeClient   kubernetes.Interface
	configClient configClientset.Interface
}

func newFLBDisableCmd(out io.Writer) *cobra.Command {
	disableCmd := &flbDisableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "disable fsm FLB",
		Long:  flbDisableDescription,
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

	return cmd
}

func (cmd *flbDisableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if !mc.Spec.FLB.Enabled {
		fmt.Fprintf(cmd.out, "FLB is disabled already, not action needed")
		return nil
	}

	if err := updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"flb.enabled": false,
	}); err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.FLB.Enabled = false
	if _, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{}); err != nil {
		return err
	}

	if err := restartFSMControllerContainer(ctx, cmd.kubeClient, fsmNamespace); err != nil {
		return err
	}

	debug("Deleting FSM FLB resources ...")
	if err := deleteFLBResources(ctx, cmd.kubeClient); err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "service-lb is disabled successfully\n")

	return nil
}
