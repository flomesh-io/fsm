// Package main implements the main entrypoint for fsm-eureka-connector and utility routines to
// bootstrap the various internal components of fsm-eureka-connector.
package main

import (
	"context"
	"net/http"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/hudl/fargo"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

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
	log    = logger.New("fsm-eureka-connector")
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting fsm-eureka-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := cli.ParseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	if err := logger.SetLogLevel(cli.Cfg.Verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", cli.Cfg.KubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", cli.Cfg.KubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	gatewayClient := gwapi.NewForConfigOrDie(kubeConfig)

	k8s.SetTrustDomain(cli.Cfg.TrustDomain)

	ctok.EnabledGatewayAPI(cli.Cfg.C2K.FlagWithGatewayAPI)
	ctok.SetSyncCloudNamespace(cli.Cfg.DeriveNamespace)

	// Initialize the generic Kubernetes event recorder and associate it with the fsm-eureka-connector pod resource
	connectorPod, err := cli.GetConnectorPod(kubeClient)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingEurekaConnectorPod)).
			Msg("Error retrieving fsm-eureka-connector pod")
		log.Fatal().Msg("Error fetching fsm-eureka-connector pod")
	}

	eventRecorder := events.GenericEventRecorder()
	if err = eventRecorder.Initialize(connectorPod, kubeClient, cli.Cfg.FsmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err = cli.ValidateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := signals.RegisterExitHandlers(cancel)

	msgBroker := messaging.NewBroker(stop)

	eurekaClient := fargo.NewConn(cli.Cfg.HttpAddr)

	sink := ctok.NewSink(ctx, kubeClient, gatewayClient, cli.Cfg.FsmNamespace)
	source := &ctok.Source{
		DiscClient:  provider.GetEurekaDiscoveryClient(&eurekaClient),
		Domain:      cli.Cfg.TrustDomain,
		Sink:        sink,
		Prefix:      "",
		FilterTag:   cli.Cfg.C2K.FlagFilterTag,
		PrefixTag:   cli.Cfg.C2K.FlagPrefixTag,
		SuffixTag:   cli.Cfg.C2K.FlagSuffixTag,
		PassingOnly: cli.Cfg.C2K.FlagPassingOnly,
	}
	sink.MicroAggregator = source
	go source.Run(ctx)

	// Build the controller and start it
	ctl := &ctok.Controller{
		Resource: sink,
	}
	go ctl.Run(ctx.Done())

	version.SetMetric()
	/*
	 * Initialize fsm-eureka-connector's HTTP server
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
	log.Info().Msgf("Stopping fsm-eureka-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}
