// Package main implements fsm xnetmgmt.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	xnetworkClientset "github.com/flomesh-io/fsm/pkg/gen/client/xnetwork/clientset/versioned"
	xnetworkscheme "github.com/flomesh-io/fsm/pkg/gen/client/xnetwork/clientset/versioned/scheme"
	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/httpserver"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
	"github.com/flomesh-io/fsm/pkg/service"
	sidecarv2 "github.com/flomesh-io/fsm/pkg/sidecar/v2"
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/flomesh-io/fsm/pkg/xnetwork"
)

var (
	verbosity         string
	meshName          string // An ID that uniquely identifies an FSM instance
	kubeConfigFile    string
	fsmNamespace      string
	fsmMeshConfigName string
	fsmVersion        string
	trustDomain       string
	nodeName          string

	enableMesh        bool
	enableE4lb        bool
	cniIPv4BridgeName string
	cniIPv6BridgeName string

	scheme = runtime.NewScheme()

	flags = pflag.NewFlagSet(`fsm-xnetmgmt`, pflag.ExitOnError)
	log   = logger.New("fsm-xnetmgmt/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&fsmNamespace, "fsm-namespace", "", "Namespace to which FSM belongs to.")
	flags.StringVar(&fsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	flags.StringVar(&fsmVersion, "fsm-version", "", "Version of FSM")
	flags.StringVar(&trustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")
	flags.StringVar(&nodeName, "node-name", os.Getenv("NODE_NAME"), "name of this Kubernetes node (spec.nodeName)")
	flags.BoolVar(&enableMesh, "enable-mesh", true, "Enable service mesh")
	flags.BoolVar(&enableE4lb, "enable-e4lb", false, "Enable 4-layer load balance")
	flags.StringVar(&cniIPv4BridgeName, "cni-ipv4-bridge-name", "", "cni ipv4 bridge name")
	flags.StringVar(&cniIPv6BridgeName, "cni-ipv6-bridge-name", "", "cni ipv6 bridge name")
	_ = clientgoscheme.AddToScheme(scheme)
	_ = xnetworkscheme.AddToScheme(scheme)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if meshName == "" {
		return fmt.Errorf("Please specify the mesh name using --mesh-name")
	}

	if fsmNamespace == "" {
		return fmt.Errorf("Please specify the FSM namespace using --fsm-namespace")
	}

	return nil
}

func main() {
	log.Info().Msgf("Starting fsm-xnetmgmt %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", kubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	configClient := configClientset.NewForConfigOrDie(kubeConfig)
	xnetworkClient := xnetworkClientset.NewForConfigOrDie(kubeConfig)

	service.SetTrustDomain(trustDomain)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := signals.RegisterExitHandlers(cancel)

	msgBroker := messaging.NewBroker(stop)

	informerCollection, err := informers.NewInformerCollection(meshName, stop,
		informers.WithKubeClient(kubeClient),
		informers.WithConfigClient(configClient, fsmMeshConfigName, fsmNamespace),
		informers.WithXNetworkClient(xnetworkClient),
	)

	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating informer collection")
	}

	kubeController := k8s.NewKubernetesController(informerCollection, nil, nil, msgBroker)

	// Initialize Configurator to watch resources in the config.flomesh.io API group
	cfg := configurator.NewConfigurator(informerCollection, fsmNamespace, fsmMeshConfigName, msgBroker)

	xnetworkController := xnetwork.NewXNetworkController(informerCollection, kubeClient, kubeController, msgBroker)

	server := sidecarv2.NewXNetConfigServer(ctx, cfg, xnetworkController, kubeClient, kubeController, xnetworkClient,
		msgBroker, enableMesh, enableE4lb, nodeName, cniIPv4BridgeName, cniIPv6BridgeName)
	server.Start()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      `xnetwork`,
			Namespace: fsmNamespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: nodeName,
		},
	}

	go func() {
		leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock:            lock,
			ReleaseOnCancel: true,
			LeaseDuration:   10 * time.Second,
			RenewDeadline:   8 * time.Second,
			RetryPeriod:     5 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					server.Leading = true
				},
				OnStoppedLeading: func() {
					server.Leading = false
				},
				OnNewLeader: func(identity string) {
					log.Info().Msgf("new leader %s", identity)
				},
			},
		})
	}()

	go server.BroadcastListener(stop)

	version.SetMetric()
	/*
	 * Initialize fsm-injector's HTTP server
	 */
	httpServer := httpserver.NewHTTPServer(constants.FSMHTTPServerPort)
	// Metrics
	httpServer.AddHandler(constants.MetricsPath, metricsstore.DefaultMetricsStore.Handler())
	// Version
	httpServer.AddHandler(constants.VersionPath, version.GetVersionHandler())
	// Health checks
	httpServer.AddHandler(constants.WebhookHealthPath, http.HandlerFunc(health.SimpleHandler))

	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start FSM metrics/probes HTTP server")
	}

	// Start the global log level watcher that updates the log level dynamically
	go k8s.WatchAndUpdateLogLevel(msgBroker, stop)

	<-stop
	cancel()
	log.Info().Msgf("Stopping fsm-xnetmgmt %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}
