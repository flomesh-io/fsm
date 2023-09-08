package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/flomesh-io/fsm/pkg/version"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/constants"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"

	"helm.sh/helm/v3/pkg/action"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/restmapper"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"k8s.io/client-go/dynamic"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const ingressEnableDescription = `
This command will enable FSM ingress, make sure --mesh-name and --fsm-namespace matches 
the release name and namespace of installed FSM, otherwise it doesn't work.
`

type ingressEnableCmd struct {
	out                     io.Writer
	kubeClient              kubernetes.Interface
	dynamicClient           dynamic.Interface
	configClient            configClientset.Interface
	nsigClient              nsigClientset.Interface
	gatewayAPIClient        gatewayApiClientset.Interface
	mapper                  meta.RESTMapper
	actionConfig            *action.Configuration
	meshName                string
	logLevel                string
	httpEnabled             bool
	httpPort                int32
	httpNodePort            int32
	tlsEnabled              bool
	mtls                    bool
	tlsPort                 int32
	tlsNodePort             int32
	passthroughEnabled      bool
	passthroughUpstreamPort int32
	replicas                int32
	serviceType             string
}

func (cmd *ingressEnableCmd) GetActionConfig() *action.Configuration {
	return cmd.actionConfig
}

func (cmd *ingressEnableCmd) GetDynamicClient() dynamic.Interface {
	return cmd.dynamicClient
}

func (cmd *ingressEnableCmd) GetRESTMapper() meta.RESTMapper {
	return cmd.mapper
}

func (cmd *ingressEnableCmd) GetMeshName() string {
	return cmd.meshName
}

func newIngressEnable(actionConfig *action.Configuration, out io.Writer) *cobra.Command {
	enableCmd := &ingressEnableCmd{
		out:          out,
		actionConfig: actionConfig,
	}

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable fsm ingress",
		Long:  ingressEnableDescription,
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

			nsigClient, err := nsigClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			enableCmd.nsigClient = nsigClient

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
	f.StringVar(&enableCmd.logLevel, "log-level", "error", "log level of ingress")
	f.BoolVar(&enableCmd.httpEnabled, "http-enable", true, "enable/disable HTTP ingress")
	f.Int32Var(&enableCmd.httpPort, "http-port", 80, "HTTP ingress port")
	f.Int32Var(&enableCmd.httpNodePort, "http-node-port", 30508, "HTTP ingress node port, take effect only if type is NodePort")
	f.BoolVar(&enableCmd.tlsEnabled, "tls-enable", false, "enable/disable TLS ingress")
	f.BoolVar(&enableCmd.mtls, "mtls", false, "enable/disable mTLS for ingress")
	f.Int32Var(&enableCmd.tlsPort, "tls-port", 443, "TLS ingress port")
	f.Int32Var(&enableCmd.tlsNodePort, "tls-node-port", 30607, "TLS ingress node port, take effect only if type is NodePort")
	f.BoolVar(&enableCmd.passthroughEnabled, "passthrough-enable", false, "enable/disable SSL passthrough")
	f.Int32Var(&enableCmd.passthroughUpstreamPort, "passthrough-upstream-port", 443, "SSL passthrough upstream port")
	f.Int32Var(&enableCmd.replicas, "replicas", 1, "replicas of ingress")
	f.StringVar(&enableCmd.serviceType, "type", string(corev1.ServiceTypeLoadBalancer), "type of ingress service, LoadBalancer or NodePort")
	//utilruntime.Must(cmd.MarkFlagRequired("mesh-name"))

	return cmd
}

func (cmd *ingressEnableCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !version.IsSupportedK8sVersion(cmd.kubeClient) {
		return fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String())
	}

	if cmd.serviceType != string(corev1.ServiceTypeLoadBalancer) && cmd.serviceType != string(corev1.ServiceTypeNodePort) {
		return fmt.Errorf("invalid service type, only support LoadBalancer or NodePort")
	}

	fsmNamespace := settings.Namespace()

	debug("Getting mesh config ...")
	// get mesh config
	mc, err := cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Get(ctx, defaultFsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// check if ingress is enabled, if yes, just return
	// TODO: check if ingress controller is installed and running???
	if mc.Spec.Ingress.Enabled {
		fmt.Fprintf(cmd.out, "Ingress is enabled, no action needed\n")
		return nil
	}

	debug("Deleting FSM NamespacedIngress resources ...")
	err = deleteNamespacedIngressResources(ctx, cmd.nsigClient)
	if err != nil {
		return err
	}

	debug("Deleting FSM Gateway resources ...")
	err = deleteGatewayResources(ctx, cmd.gatewayAPIClient)
	if err != nil {
		return err
	}

	err = updatePresetMeshConfigMap(ctx, cmd.kubeClient, fsmNamespace, map[string]interface{}{
		"ingress.enabled":                         true,
		"ingress.namespaced":                      false,
		"ingress.type":                            cmd.serviceType,
		"ingress.logLevel":                        cmd.logLevel,
		"ingress.http.enabled":                    cmd.httpEnabled,
		"ingress.http.bind":                       cmd.httpPort,
		"ingress.http.nodePort":                   cmd.httpNodePort,
		"ingress.tls.enabled":                     cmd.tlsEnabled,
		"ingress.tls.bind":                        cmd.tlsPort,
		"ingress.tls.nodePort":                    cmd.tlsNodePort,
		"ingress.tls.mTLS":                        cmd.mtls,
		"ingress.tls.sslPassthrough.enabled":      cmd.passthroughEnabled,
		"ingress.tls.sslPassthrough.upstreamPort": cmd.passthroughUpstreamPort,
		"gatewayAPI.enabled":                      false,
	})
	if err != nil {
		return err
	}

	debug("Updating mesh config ...")
	// update mesh config, fsm-mesh-config
	mc.Spec.Ingress.Enabled = true
	mc.Spec.Ingress.Namespaced = false
	mc.Spec.Ingress.Type = corev1.ServiceType(cmd.serviceType)
	mc.Spec.Ingress.LogLevel = cmd.logLevel
	mc.Spec.Ingress.HTTP.Enabled = cmd.httpEnabled
	mc.Spec.Ingress.HTTP.Bind = cmd.httpPort
	mc.Spec.Ingress.HTTP.NodePort = cmd.httpNodePort
	mc.Spec.Ingress.TLS.Enabled = cmd.tlsEnabled
	mc.Spec.Ingress.TLS.Bind = cmd.tlsPort
	mc.Spec.Ingress.TLS.NodePort = cmd.tlsNodePort
	mc.Spec.Ingress.TLS.MTLS = cmd.mtls
	mc.Spec.Ingress.TLS.SSLPassthrough.Enabled = cmd.passthroughEnabled
	mc.Spec.Ingress.TLS.SSLPassthrough.UpstreamPort = cmd.passthroughUpstreamPort
	mc.Spec.GatewayAPI.Enabled = false
	_, err = cmd.configClient.ConfigV1alpha3().MeshConfigs(fsmNamespace).Update(ctx, mc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	debug("Restarting fsm-controller ...")
	// Rollout restart fsm-controller
	// patch the deployment spec template triggers the action of rollout restart like with kubectl
	if err := restartFSMController(ctx, cmd.kubeClient, fsmNamespace, cmd.out); err != nil {
		return err
	}

	if err := installManifests(cmd, mc, fsmNamespace, kubeVersion119, ingressManifestFiles...); err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	deployment, err := cmd.kubeClient.AppsV1().
		Deployments(fsmNamespace).
		Get(ctx, constants.FSMIngressName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if err := waitForDeploymentReady(ctx, cmd.kubeClient, deployment, cmd.out); err != nil {
		return err
	}

	fmt.Fprintf(cmd.out, "Ingress is enabled successfully\n")

	return nil
}

func (cmd *ingressEnableCmd) ResolveValues(mc *configv1alpha3.MeshConfig) (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	valuesConfig := []string{
		fmt.Sprintf("fsm.fsmIngress.enabled=%t", true),
		fmt.Sprintf("fsm.fsmIngress.namespaced=%t", false),
		fmt.Sprintf("fsm.fsmIngress.service.type=%s", cmd.serviceType),
		fmt.Sprintf("fsm.fsmIngress.logLevel=%s", cmd.logLevel),
		fmt.Sprintf("fsm.fsmIngress.http.enabled=%t", cmd.httpEnabled),
		fmt.Sprintf("fsm.fsmIngress.http.port=%d", cmd.httpPort),
		fmt.Sprintf("fsm.fsmIngress.http.nodePort=%d", cmd.httpNodePort),
		fmt.Sprintf("fsm.fsmIngress.tls.enabled=%t", cmd.tlsEnabled),
		fmt.Sprintf("fsm.fsmIngress.tls.port=%d", cmd.tlsPort),
		fmt.Sprintf("fsm.fsmIngress.tls.nodePort=%d", cmd.tlsNodePort),
		fmt.Sprintf("fsm.fsmIngress.tls.mTLS=%t", cmd.mtls),
		fmt.Sprintf("fsm.fsmIngress.tls.sslPassthrough.enabled=%t", cmd.passthroughEnabled),
		fmt.Sprintf("fsm.fsmIngress.tls.sslPassthrough.upstreamPort=%d", cmd.passthroughUpstreamPort),
		fmt.Sprintf("fsm.fsmIngress.replicaCount=%d", cmd.replicas),
		fmt.Sprintf("fsm.fsmGateway.enabled=%t", false),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetNamespace()),
		fmt.Sprintf("fsm.meshName=%s", cmd.meshName),
		fmt.Sprintf("fsm.image.registry=%s", mc.Spec.Image.Registry),
		fmt.Sprintf("fsm.image.pullPolicy=%s", mc.Spec.Image.PullPolicy),
		fmt.Sprintf("fsm.image.tag=%s", mc.Spec.Image.Tag),
		fmt.Sprintf("fsm.curlImage=%s", mc.Spec.Misc.CurlImage),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return nil, err
	}

	return finalValues, nil
}
