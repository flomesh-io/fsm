// Package main implements the main entrypoint for fsm-connector and utility routines to
// bootstrap the various internal components of fsm-connector.
package main

import (
	"context"
	"net/http"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/cli"
	"github.com/flomesh-io/fsm/pkg/connector/ctok"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/httpserver"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	_ "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/driver"
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/version"
)

var (
	log    = logger.New("fsm-connector")
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting fsm-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := cli.ParseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := cli.ValidateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	if err := logger.SetLogLevel(cli.Verbosity()); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", cli.Cfg.KubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", cli.Cfg.KubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	// Initialize the generic Kubernetes event recorder and associate it with the fsm-connector pod resource
	connectorPod, err := cli.GetConnectorPod(kubeClient)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingConnectorPod)).
			Msg("Error retrieving fsm-connector pod")
		log.Fatal().Msg("Error fetching fsm-connector pod")
	}

	eventRecorder := events.GenericEventRecorder()
	if err = eventRecorder.Initialize(connectorPod, kubeClient, cli.Cfg.FsmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	k8s.SetTrustDomain(cli.Cfg.TrustDomain)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := signals.RegisterExitHandlers(cancel)

	msgBroker := messaging.NewBroker(stop)

	var discClient provider.ServiceDiscoveryClient = nil
	if connector.EurekaDiscoveryService == cli.Cfg.SdrProvider {
		discClient, err = provider.GetEurekaDiscoveryClient(cli.Cfg.HttpAddr)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating cloud client")
			log.Fatal().Msg("Error creating cloud client")
		}
	} else if connector.ConsulDiscoveryService == cli.Cfg.SdrProvider {
		discClient, err = provider.GetConsulDiscoveryClient(cli.Cfg.HttpAddr)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating cloud client")
			log.Fatal().Msg("Error creating cloud client")
		}
	} else {
		log.Fatal().Msg("Unsupported service discovery and registration provider")
	}

	gatewayClient := gwapi.NewForConfigOrDie(kubeConfig)

	go cli.SyncCtoK(ctx, kubeClient, discClient, gatewayClient)
	go cli.SyncKtoC(ctx, kubeClient, discClient)

	version.SetMetric()
	/*
	 * Initialize fsm-connector's HTTP server
	 */
	httpServer := httpserver.NewHTTPServer(constants.FSMHTTPServerPort)
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
	go ctok.WatchMeshConfigUpdated(msgBroker, stop)

	<-stop
	log.Info().Msgf("Stopping fsm-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}
