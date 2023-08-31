package main

import (
	"context"
	"fmt"
	"io"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
)

const namespacedIngressEnableDescription = `
This command will enable FSM NamespacedIngress, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type namespacedIngressEnableCmd struct {
	out              io.Writer
	kubeClient       kubernetes.Interface
	configClient     configClientset.Interface
	gatewayAPIClient gatewayApiClientset.Interface
	meshName         string
	version          string
}

func newNamespacedIngressEnableCmd(out io.Writer) *cobra.Command {
	enableCmd := &namespacedIngressEnableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable fsm NamespacedIngress",
		Long:  namespacedIngressEnableDescription,
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

			configClient, err := configClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			enableCmd.configClient = configClient

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

func (cmd *namespacedIngressEnableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if mc.Spec.Ingress.Enabled && mc.Spec.Ingress.Namespaced {
		fmt.Fprintf(cmd.out, "NamespacedIngress is enabled already, not action needed")
		return nil
	}

	debug("Deleting FSM Ingress resources ...")
	err = deleteIngressResources(ctx, cmd.kubeClient, fsmNamespace, cmd.meshName)
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
		"ingress.namespaced": true,
		"gatewayAPI.enabled": false,
	})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.Ingress.Enabled = true
	mc.Spec.Ingress.Namespaced = true
	mc.Spec.GatewayAPI.Enabled = false
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	err = restartFSMControllerContainer(ctx, cmd.kubeClient, fsmNamespace)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "NamespacedIngress is enabled successfully\n")

	return nil
}
