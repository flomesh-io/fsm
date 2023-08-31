package main

import (
	"context"
	"fmt"
	"io"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"k8s.io/client-go/restmapper"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"

	"helm.sh/helm/v3/pkg/action"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
)

const flbEnableDescription = `
This command will enable FSM FLB, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type flbEnableCmd struct {
	out           io.Writer
	kubeClient    kubernetes.Interface
	configClient  configClientset.Interface
	dynamicClient dynamic.Interface
	mapper        meta.RESTMapper
	actionConfig  *action.Configuration
	meshName      string
	strictMode    bool
	secretName    string
	baseUrl       string
	userName      string
	password      string
	k8sCluster    string
	cluster       string
	addressPool   string
	algo          string
}

func (cmd *flbEnableCmd) GetActionConfig() *action.Configuration {
	return cmd.actionConfig
}

func (cmd *flbEnableCmd) GetDynamicClient() dynamic.Interface {
	return cmd.dynamicClient
}

func (cmd *flbEnableCmd) GetRESTMapper() meta.RESTMapper {
	return cmd.mapper
}

func (cmd *flbEnableCmd) GetMeshName() string {
	return cmd.meshName
}

func newFLBEnableCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	enableCmd := &flbEnableCmd{
		out:          out,
		actionConfig: config,
	}

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable fsm FLB",
		Long:  flbEnableDescription,
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

			return enableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&enableCmd.meshName, "mesh-name", defaultMeshName, "name for the control plane instance")
	f.BoolVar(&enableCmd.strictMode, "strict-mode", false, "enable strict mode for FLB")
	f.StringVar(&enableCmd.secretName, "secret-name", "fsm-flb-secret", "name of the secret for storing FLB config")
	f.StringVar(&enableCmd.baseUrl, "base-url", "http://localhost:1337", "base URL of FLB API server")
	f.StringVar(&enableCmd.userName, "username", "admin", "user name of FLB API server")
	f.StringVar(&enableCmd.password, "password", "admin", "password of FLB API server")
	f.StringVar(&enableCmd.k8sCluster, "k8s-cluster", "UNKNOWN", "name of the k8s cluster in which FLB controller is running")
	f.StringVar(&enableCmd.cluster, "cluster", "default", "name of the pipy cluster of FLB data plane")
	f.StringVar(&enableCmd.addressPool, "address-pool", "default", "name of the address pool of FLB")
	f.StringVar(&enableCmd.algo, "algo", "default", "load balancing algorithm of FLB")

	//utilruntime.Must(cmd.MarkFlagRequired("mesh-name"))

	return cmd
}

func (cmd *flbEnableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if mc.Spec.ServiceLB.Enabled {
		fmt.Fprintf(cmd.out, "service-lb is enabled already, not action needed")
		return nil
	}

	err = updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"flb.enabled":    true,
		"flb.strictMode": cmd.strictMode,
		"flb.secretName": cmd.secretName,
	})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.FLB.Enabled = true
	mc.Spec.FLB.StrictMode = cmd.strictMode
	mc.Spec.FLB.SecretName = cmd.secretName
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if err := installManifests(cmd, mc, fsmNamespace, kubeVersion119, flbManifestFiles...); err != nil {
		return err
	}

	err = restartFSMControllerContainer(ctx, cmd.kubeClient, fsmNamespace)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "FLB is enabled successfully\n")

	return nil
}

func (cmd *flbEnableCmd) ResolveValues(mc *configv1alpha3.MeshConfig) (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	valuesConfig := []string{
		fmt.Sprintf("fsm.flb.enabled=%t", true),
		fmt.Sprintf("fsm.flb.strictMode=%t", cmd.strictMode),
		fmt.Sprintf("fsm.flb.secretName=%s", cmd.secretName),
		fmt.Sprintf("fsm.flb.baseUrl=%s", cmd.baseUrl),
		fmt.Sprintf("fsm.flb.password=%s", cmd.password),
		fmt.Sprintf("fsm.flb.k8sCluster=%s", cmd.kubeClient),
		fmt.Sprintf("fsm.flb.defaultCluster=%s", cmd.cluster),
		fmt.Sprintf("fsm.flb.defaultAddressPool=%s", cmd.addressPool),
		fmt.Sprintf("fsm.flb.defaultAlgo=%s", cmd.algo),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetNamespace()),
		fmt.Sprintf("fsm.meshName=%s", cmd.meshName),
		fmt.Sprintf("fsm.image.registry=%s", mc.Spec.Image.Registry),
		fmt.Sprintf("fsm.image.pullPolicy=%s", mc.Spec.Image.PullPolicy),
		fmt.Sprintf("fsm.image.tag=%s", mc.Spec.Image.Tag),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return nil, err
	}

	return finalValues, nil
}
