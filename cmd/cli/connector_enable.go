package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/flomesh-io/fsm/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
)

const connectorEnableDescription = `
This command will enable FSM connector, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type connectorEnableCmd struct {
	out           io.Writer
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
	configClient  configClientset.Interface
	mapper        meta.RESTMapper
	actionConfig  *action.Configuration
	meshName      string
	setOptions    []string // --set
	timeout       time.Duration

	connectorManifestFiles []string
	connectorDeployments   []string
}

func (cmd *connectorEnableCmd) GetActionConfig() *action.Configuration {
	return cmd.actionConfig
}

func (cmd *connectorEnableCmd) GetDynamicClient() dynamic.Interface {
	return cmd.dynamicClient
}

func (cmd *connectorEnableCmd) GetRESTMapper() meta.RESTMapper {
	return cmd.mapper
}

func (cmd *connectorEnableCmd) GetMeshName() string {
	return cmd.meshName
}

func newConnectorEnable(actionConfig *action.Configuration, out io.Writer) *cobra.Command {
	enableCmd := &connectorEnableCmd{
		out:          out,
		actionConfig: actionConfig,
	}

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable fsm connector",
		Long:  connectorEnableDescription,
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
	f.DurationVar(&enableCmd.timeout, "timeout", 5*time.Minute, "Time to wait for installation and resources in a ready state, zero means no timeout")
	f.StringArrayVar(&enableCmd.setOptions, "set", nil, "Set arbitrary chart values (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	return cmd
}

func (cmd *connectorEnableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if err := installManifests(cmd, mc, fsmNamespace, constants.KubeVersion119, cmd.connectorManifestFiles...); err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	var wg sync.WaitGroup
	for _, deploy := range cmd.connectorDeployments {
		wg.Add(1)
		go func(deploy string) {
			defer wg.Done()
			deployment, err := cmd.kubeClient.AppsV1().
				Deployments(fsmNamespace).
				Get(ctx, deploy, metav1.GetOptions{})
			if err != nil {
				return
			}

			if err := waitForDeploymentReady(ctx, cmd.kubeClient, deployment, cmd.out); err != nil {
				return
			}

			fmt.Fprintf(cmd.out, fmt.Sprintf("%s is enabled successfully\n", deploy))
		}(deploy)
	}
	wg.Wait()

	return nil
}

func (cmd *connectorEnableCmd) ResolveValues(mc *configv1alpha3.MeshConfig, manifestFiles ...string) ([]string, map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	valuesConfig := []string{
		fmt.Sprintf("fsm.meshName=%s", cmd.meshName),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetNamespace()),
		fmt.Sprintf("fsm.controllerLogLevel=%s", mc.Spec.Observability.FSMLogLevel),
		fmt.Sprintf("fsm.image.registry=%s", mc.Spec.Image.Registry),
		fmt.Sprintf("fsm.image.pullPolicy=%s", mc.Spec.Image.PullPolicy),
		fmt.Sprintf("fsm.image.tag=%s", mc.Spec.Image.Tag),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return manifestFiles, nil, err
	}

	if err := parseVal(cmd.setOptions, finalValues); err != nil {
		return manifestFiles, nil, fmt.Errorf("invalid format for --set: %w", err)
	}

	cmd.parseConnectorFinalValues(finalValues)

	manifestFiles = cmd.connectorManifestFiles

	if len(manifestFiles) == 0 {
		return manifestFiles, nil, errors.New("no connector was enabled")
	}

	return manifestFiles, finalValues, nil
}

func (cmd *connectorEnableCmd) parseConnectorFinalValues(finalValues map[string]interface{}) {
	var enablePodDisruptionBudget, enableAutoScale bool
	if v, ok := finalValues["fsm"]; ok {
		fsm := v.(map[string]interface{})
		if v, ok = fsm["cloudConnector"]; ok {
			cloudConnector := v.(map[string]interface{})
			if v, ok = cloudConnector["enablePodDisruptionBudget"]; ok {
				enablePodDisruptionBudget = v.(bool)
			}
			if v, ok = cloudConnector["autoScale"]; ok {
				autoScale := v.(map[string]interface{})
				if v, ok = autoScale["enable"]; ok {
					enableAutoScale = v.(bool)
				}
			}
			cmd.parseConsulConnectorFinalValues(cloudConnector, enablePodDisruptionBudget, enableAutoScale)
			cmd.parseEurekaConnectorFinalValues(cloudConnector, enablePodDisruptionBudget, enableAutoScale)
			cmd.parseNacosConnectorFinalValues(cloudConnector, enablePodDisruptionBudget, enableAutoScale)
			cmd.parseMachineConnectorFinalValues(cloudConnector, enablePodDisruptionBudget, enableAutoScale)
			cmd.parseGatewayConnectorFinalValues(cloudConnector, enablePodDisruptionBudget, enableAutoScale)
		}
	}
}

func (cmd *connectorEnableCmd) parseConsulConnectorFinalValues(cloudConnector map[string]interface{}, enablePodDisruptionBudget bool, enableAutoScale bool) {
	enableConsul := false
	if v, ok := cloudConnector["consul"]; ok {
		consul := v.(map[string]interface{})
		if v, ok = consul["enable"]; ok {
			enableConsul = v.(bool)
		}
		if enableConsul {
			suffix := ""
			if v, ok = consul["connectorNameSuffix"]; ok {
				suffix = v.(string)
			}
			if len(suffix) == 0 {
				suffix = "consul"
			}
			cmd.connectorDeployments = append(cmd.connectorDeployments, fmt.Sprintf("fsm-connector-%s", suffix))
			cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-consul-deployment.yaml")
			if enablePodDisruptionBudget {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-consul-pod-disruption-budget.yaml")
			}
			if enableAutoScale {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-consul-hpa.yaml")
			}
		}
	}
}

func (cmd *connectorEnableCmd) parseEurekaConnectorFinalValues(cloudConnector map[string]interface{}, enablePodDisruptionBudget bool, enableAutoScale bool) {
	enableEureka := false
	if v, ok := cloudConnector["eureka"]; ok {
		eureka := v.(map[string]interface{})
		if v, ok = eureka["enable"]; ok {
			enableEureka = v.(bool)
		}
		if enableEureka {
			suffix := ""
			if v, ok = eureka["connectorNameSuffix"]; ok {
				suffix = v.(string)
			}
			if len(suffix) == 0 {
				suffix = "eureka"
			}
			cmd.connectorDeployments = append(cmd.connectorDeployments, fmt.Sprintf("fsm-connector-%s", suffix))
			cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-eureka-deployment.yaml")
			if enablePodDisruptionBudget {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-eureka-pod-disruption-budget.yaml")
			}
			if enableAutoScale {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-eureka-hpa.yaml")
			}
		}
	}
}

func (cmd *connectorEnableCmd) parseNacosConnectorFinalValues(cloudConnector map[string]interface{}, enablePodDisruptionBudget bool, enableAutoScale bool) {
	enableNacos := false
	if v, ok := cloudConnector["nacos"]; ok {
		nacos := v.(map[string]interface{})
		if v, ok = nacos["enable"]; ok {
			enableNacos = v.(bool)
		}
		if enableNacos {
			suffix := ""
			if v, ok = nacos["connectorNameSuffix"]; ok {
				suffix = v.(string)
			}
			if len(suffix) == 0 {
				suffix = "nacos"
			}
			cmd.connectorDeployments = append(cmd.connectorDeployments, fmt.Sprintf("fsm-connector-%s", suffix))
			cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-nacos-deployment.yaml")
			if enablePodDisruptionBudget {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-nacos-pod-disruption-budget.yaml")
			}
			if enableAutoScale {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-nacos-hpa.yaml")
			}
		}
	}
}

func (cmd *connectorEnableCmd) parseMachineConnectorFinalValues(cloudConnector map[string]interface{}, enablePodDisruptionBudget bool, enableAutoScale bool) {
	enableMachine := false
	if v, ok := cloudConnector["machine"]; ok {
		machine := v.(map[string]interface{})
		if v, ok = machine["enable"]; ok {
			enableMachine = v.(bool)
		}
		if enableMachine {
			suffix := ""
			if v, ok = machine["connectorNameSuffix"]; ok {
				suffix = v.(string)
			}
			if len(suffix) == 0 {
				suffix = "machine"
			}
			cmd.connectorDeployments = append(cmd.connectorDeployments, fmt.Sprintf("fsm-connector-%s", suffix))
			cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-machine-deployment.yaml")
			if enablePodDisruptionBudget {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-machine-pod-disruption-budget.yaml")
			}
			if enableAutoScale {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-machine-hpa.yaml")
			}
		}
	}
}

func (cmd *connectorEnableCmd) parseGatewayConnectorFinalValues(cloudConnector map[string]interface{}, enablePodDisruptionBudget bool, enableAutoScale bool) {
	enableGateway := false
	if v, ok := cloudConnector["gateway"]; ok {
		gateway := v.(map[string]interface{})
		if v, ok = gateway["syncToFgw"]; ok {
			syncToFgw := v.(map[string]interface{})
			if v, ok = syncToFgw["enable"]; ok {
				enableGateway = v.(bool)
			}
		}
		if enableGateway {
			suffix := ""
			if v, ok = gateway["connectorNameSuffix"]; ok {
				suffix = v.(string)
			}
			if len(suffix) == 0 {
				suffix = "gateway"
			}
			cmd.connectorDeployments = append(cmd.connectorDeployments, fmt.Sprintf("fsm-connector-%s", suffix))
			cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-gateway-deployment.yaml")
			if enablePodDisruptionBudget {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-gateway-pod-disruption-budget.yaml")
			}
			if enableAutoScale {
				cmd.connectorManifestFiles = append(cmd.connectorManifestFiles, "templates/fsm-connector-gateway-hpa.yaml")
			}
		}
	}
}
