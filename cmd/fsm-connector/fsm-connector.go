// Package main implements the main entrypoint for fsm-connector and utility routines to
// bootstrap the various internal components of fsm-connector.
package main

import (
	"context"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gwscheme "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/scheme"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/cli"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	connectorClientset "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned"
	connectorscheme "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned/scheme"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	machinescheme "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned/scheme"
	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/httpserver"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/version"
)

var (
	log    = logger.New("fsm-connector")
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = gwscheme.AddToScheme(scheme)
	_ = machinescheme.AddToScheme(scheme)
	_ = connectorscheme.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting fsm-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := cli.ParseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := cli.ValidateCLIParams(); err != nil {
		log.Fatal().Err(err).Msg("Error validating CLI parameters")
	}

	if err := logger.SetLogLevel(cli.Verbosity()); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", cli.Cfg.KubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", cli.Cfg.KubeConfigFile)
	}
	kubeConfig.QPS = float32(cli.Cfg.Limit)
	kubeConfig.Burst = int(cli.Cfg.Burst)
	kubeConfig.Timeout = time.Second * time.Duration(cli.Cfg.Timeout)
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	machineClient := machineClientset.NewForConfigOrDie(kubeConfig)
	gatewayClient := gwapi.NewForConfigOrDie(kubeConfig)
	connectorClient := connectorClientset.NewForConfigOrDie(kubeConfig)
	gatewayApiClient := gatewayApiClientset.NewForConfigOrDie(kubeConfig)

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

	service.SetTrustDomain(cli.Cfg.TrustDomain)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := signals.RegisterExitHandlers(cancel)

	msgBroker := messaging.NewBroker(stop)
	configClient := configClientset.NewForConfigOrDie(kubeConfig)
	informerCollection, err := informers.NewInformerCollection(cli.Cfg.MeshName, stop,
		informers.WithConfigClient(configClient, cli.Cfg.FsmMeshConfigName, cli.Cfg.FsmNamespace),
		informers.WithMachineClient(machineClient),
		informers.WithConnectorClient(connectorClient),
		informers.WithGatewayAPIClient(gatewayApiClient),
	)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating informer collection")
	}
	cfg := configurator.NewConfigurator(informerCollection, cli.Cfg.FsmNamespace, cli.Cfg.FsmMeshConfigName, msgBroker)
	connectController := cli.NewConnectController(
		cli.Cfg.SdrProvider, cli.Cfg.SdrConnector,
		ctx, kubeConfig, kubeClient, configClient,
		connectorClient, machineClient, gatewayClient,
		informerCollection, msgBroker)

	connector.GatewayAPIEnabled = cfg.GetMeshConfig().Spec.GatewayAPI.Enabled
	clusterSet := cfg.GetMeshConfig().Spec.ClusterSet
	connectController.SetClusterSet(clusterSet.Name, clusterSet.Group, clusterSet.Zone, clusterSet.Region)

	if cli.Cfg.LeaderElection {
		lock := &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Name:      cli.Cfg.SdrConnector,
				Namespace: connectorPod.Namespace,
			},
			Client: kubeClient.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: connectorPod.Name,
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
						go connectController.BroadcastListener(stop)
						go connectController.CacheCleaner(stop)
					},
					OnStoppedLeading: func() {
					},
					OnNewLeader: func(identity string) {
						log.Info().Msgf("new leader %s", identity)
					},
				},
			})
		}()
	} else {
		go connectController.BroadcastListener(stop)
		go connectController.CacheCleaner(stop)
	}

	// Start the global log level watcher that updates the log level dynamically
	go connector.WatchMeshConfigUpdated(connectController, msgBroker, stop)

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

	<-stop
	log.Info().Msgf("Stopping fsm-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}
