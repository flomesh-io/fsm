// Package main contains the main function for the fsm-ingress binary
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/httpserver"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/version"
)

type metadata struct {
	PodName      string `envconfig:"POD_NAME" required:"true" split_words:"true"`
	PodNamespace string `envconfig:"POD_NAMESPACE" required:"true" split_words:"true"`
}

var (
	flags = pflag.NewFlagSet(`fsm-ingress`, pflag.ExitOnError)
	log   = logger.New("fsm-ingress/main")
)

var (
	verbosity         string
	meshName          string // An ID that uniquely identifies an FSM instance
	fsmNamespace      string
	fsmMeshConfigName string
	fsmVersion        string

	meta metadata
)

const (
	httpSchema = "http"
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", constants.DefaultFSMLogLevel, "Set boot log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&fsmNamespace, "fsm-namespace", "", "FSM controller's namespace")
	flags.StringVar(&fsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	flags.StringVar(&fsmVersion, "fsm-version", "", "Version of FSM")

	meta = getMetadata()
}

func getMetadata() metadata {
	var metadata metadata

	err := envconfig.Process("FSM", &metadata)
	if err != nil {
		log.Error().Msgf("unable to load FSM metadata from environment: %s", err)
		panic(err)
	}

	return metadata
}

func main() {
	log.Info().Msgf("Starting fsm-ingress %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Str(errcode.Kind, errcode.ErrInvalidCLIArgument.String()).Msg("Error parsing cmd line arguments")
	}

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	kubeconfig := ctrl.GetConfigOrDie()
	kubeClient := kubernetes.NewForConfigOrDie(kubeconfig)
	configClient := configClientset.NewForConfigOrDie(kubeconfig)

	if !version.IsSupportedK8sVersionForGatewayAPI(kubeClient) {
		log.Error().Msgf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersionForGatewayAPI.String())
		os.Exit(1)
	}

	_, cancel := context.WithCancel(context.TODO())
	stop := signals.RegisterExitHandlers(cancel)
	msgBroker := messaging.NewBroker(stop)

	informerCollection, err := informers.NewInformerCollection(meshName, stop,
		//informers.WithKubeClient(kubeClient),
		informers.WithConfigClient(configClient, fsmMeshConfigName, fsmNamespace),
	)
	if err != nil {
		log.Error().Msgf("")
	}

	cfg := configurator.NewConfigurator(informerCollection, fsmNamespace, fsmMeshConfigName, msgBroker)

	if !cfg.IsIngressEnabled() {
		log.Error().Msgf("Ingress is not enabled, FSM doesn't support Ingress and GatewayAPI are both enabled.")
		os.Exit(1)
	}

	// get ingress codebase
	ingressRepoURL := ingressCodebase(cfg)
	log.Info().Msgf("Ingress Repo = %q", ingressRepoURL)

	// calculate pipy spawn
	spawn := calcPipySpawn(kubeClient)
	log.Info().Msgf("PIPY SPAWN = %d", spawn)

	// start pipy
	startPipy(spawn, ingressRepoURL)

	startHTTPServer()

	<-stop
	cancel()
	log.Info().Msgf("Stopping fsm-ingress %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

func ingressCodebase(cfg configurator.Configurator) string {
	repoHost := fmt.Sprintf("%s.%s.svc", constants.FSMControllerName, fsmNamespace)
	repoPort := cfg.GetProxyServerPort()

	if cfg.IsNamespacedIngressEnabled() {
		return fmt.Sprintf("%s://%s:%d/repo%s/", httpSchema, repoHost, repoPort, utils.NamespacedIngressCodebasePath(meta.PodNamespace))
	}
	if cfg.IsIngressEnabled() {
		return fmt.Sprintf("%s://%s:%d/repo%s/", httpSchema, repoHost, repoPort, utils.IngressCodebasePath())
	}

	return ""
}

func calcPipySpawn(kubeClient kubernetes.Interface) int64 {
	cpuLimits, err := getIngressCPULimitsQuota(kubeClient)
	if err != nil {
		log.Fatal().Err(err)
		os.Exit(1)
	}
	log.Info().Msgf("CPU Limits = %v", cpuLimits)

	spawn := int64(1)
	if cpuLimits.Value() > 0 {
		spawn = cpuLimits.Value()
	}

	return spawn
}

func getIngressPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podNamespace := meta.PodNamespace
	podName := meta.PodName

	pod, err := kubeClient.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error().Msgf("Error retrieving ingress-pipy pod %s", podName)
		return nil, err
	}

	return pod, nil
}

func getIngressCPULimitsQuota(kubeClient kubernetes.Interface) (*resource.Quantity, error) {
	pod, err := getIngressPod(kubeClient)
	if err != nil {
		return nil, err
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == "ingress" {
			return c.Resources.Limits.Cpu(), nil
		}
	}

	return nil, errors.Errorf("No container named 'ingress' in POD %q", pod.Name)
}

func startPipy(spawn int64, ingressRepoURL string) {
	args := []string{ingressRepoURL}
	if spawn > 1 {
		args = append([]string{"--reuse-port", fmt.Sprintf("--threads=%d", spawn)}, args...)
	}
	if verbosity != "disabled" {
		args = append([]string{fmt.Sprintf("--log-level=%s", utils.PipyLogLevelByVerbosity(verbosity))}, args...)
	}

	cmd := exec.Command("pipy", args...) // #nosec G204
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info().Msgf("cmd = %v", cmd)

	if err := cmd.Start(); err != nil {
		log.Fatal().Err(err)
		os.Exit(1)
	}
}

func startHTTPServer() {
	// Initialize FSM's http service server
	httpServer := httpserver.NewHTTPServer(constants.FSMHTTPServerPort)
	// Metrics
	httpServer.AddHandler(constants.MetricsPath, metricsstore.DefaultMetricsStore.Handler())
	// Version
	httpServer.AddHandler(constants.VersionPath, version.GetVersionHandler())

	// Start HTTP server
	err := httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start FSM Ingress metrics/probes HTTP server")
		os.Exit(1)
	}
}
