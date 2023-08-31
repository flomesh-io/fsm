package main

import (
	"context"
	"fmt"
	"io"

	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const gatewayEnableDescription = `
This command will enable FSM gateway, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type gatewayEnableCmd struct {
	out          io.Writer
	kubeClient   kubernetes.Interface
	configClient configClientset.Interface
	nsigClient   nsigClientset.Interface
	meshName     string
	logLevel     string
}

func newGatewayEnable(out io.Writer) *cobra.Command {
	enableCmd := &gatewayEnableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable fsm gateway",
		Long:  gatewayEnableDescription,
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

			nsigClient, err := nsigClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			enableCmd.nsigClient = nsigClient

			return enableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&enableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")
	f.StringVar(&enableCmd.logLevel, "log-level", "error", "log level of gateway")
	//utilruntime.Must(cmd.MarkFlagRequired("mesh-name"))

	return cmd
}

func (cmd *gatewayEnableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// check if gateway is enabled, if yes, just return
	// TODO: check if GatewayClass is installed and if there's any running gateway instances
	if mc.Spec.GatewayAPI.Enabled {
		fmt.Fprintf(cmd.out, "Gatweway is enabled already, not action needed")
		return nil
	}

	debug("Deleting FSM Ingress resources ...")
	err = deleteIngressResources(ctx, cmd.kubeClient, fsmNamespace, cmd.meshName)
	if err != nil {
		return err
	}

	debug("Deleting FSM NamespacedIngress resources ...")
	err = deleteNamespacedIngressResources(ctx, cmd.nsigClient)
	if err != nil {
		return err
	}

	err = updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"ingress.enabled":     false,
		"ingress.namespaced":  false,
		"gatewayAPI.enabled":  true,
		"gatewayAPI.logLevel": cmd.logLevel,
	})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.Ingress.Enabled = false
	mc.Spec.Ingress.Namespaced = false
	mc.Spec.GatewayAPI.Enabled = true
	mc.Spec.GatewayAPI.LogLevel = cmd.logLevel
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	err = restartFSMControllerContainer(ctx, cmd.kubeClient, fsmNamespace)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "Gateway is enabled successfully\n")

	return nil
}
