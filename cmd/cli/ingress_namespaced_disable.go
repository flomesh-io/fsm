package main

import (
	"context"
	"fmt"
	"io"

	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
)

const namespacedIngressDisableDescription = `
This command will disable FSM NamespacedIngress, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type namespacedIngressDisableCmd struct {
	out          io.Writer
	kubeClient   kubernetes.Interface
	configClient configClientset.Interface
	nsigClient   nsigClientset.Interface
	meshName     string
}

func newNamespacedIngressDisableCmd(out io.Writer) *cobra.Command {
	disableCmd := &namespacedIngressDisableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "disable fsm NamespacedIngress",
		Long:  namespacedIngressDisableDescription,
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

			nsigClient, err := nsigClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			disableCmd.nsigClient = nsigClient

			return disableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&disableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")
	//utilruntime.Must(cmd.MarkFlagRequired("mesh-name"))

	return cmd
}

func (cmd *namespacedIngressDisableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if !mc.Spec.Ingress.Enabled && !mc.Spec.Ingress.Namespaced {
		fmt.Fprintf(cmd.out, "NamespacedIngress is disabled already, not action needed")
		return nil
	}

	debug("Deleting FSM NamespacedIngress resources ...")
	err = deleteNamespacedIngressResources(ctx, cmd.nsigClient)
	if err != nil {
		return err
	}

	err = updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"ingress.enabled":    false,
		"ingress.namespaced": false,
	})
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

	err = restartFSMControllerContainer(ctx, cmd.kubeClient, fsmNamespace)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "NamespacedIngress is disabled successfully\n")

	return nil
}
