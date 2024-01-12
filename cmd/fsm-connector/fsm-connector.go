// Package main implements the main entrypoint for fsm-connector and utility routines to
// bootstrap the various internal components of fsm-connector.
package main

import (
	"context"
	"fmt"
	"net/http"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/cli"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/httpserver"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
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
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	machineClient := machineClientset.NewForConfigOrDie(kubeConfig)

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
	configClient := configClientset.NewForConfigOrDie(kubeConfig)
	informerCollection, err := informers.NewInformerCollection(cli.Cfg.MeshName, stop,
		informers.WithKubeClient(kubeClient),
		informers.WithConfigClient(configClient, cli.Cfg.FsmMeshConfigName, cli.Cfg.FsmNamespace),
		informers.WithMachineClient(machineClient),
	)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating informer collection")
	}
	cfg := configurator.NewConfigurator(informerCollection, cli.Cfg.FsmNamespace, cli.Cfg.FsmMeshConfigName, msgBroker)
	connector.GatewayAPIEnabled = cfg.GetMeshConfig().Spec.GatewayAPI.Enabled
	clusterSet := cfg.GetMeshConfig().Spec.ClusterSet
	connector.ServiceSourceValue = fmt.Sprintf("%s.%s.%s.%s", clusterSet.Name, clusterSet.Group, clusterSet.Zone, clusterSet.Region)

	if len(cli.Cfg.SdrProvider) > 0 {
		var discClient provider.ServiceDiscoveryClient = nil
		if connector.EurekaDiscoveryService == cli.Cfg.SdrProvider {
			discClient, err = provider.GetEurekaDiscoveryClient(cli.Cfg.HttpAddr, cli.Cfg.AsInternalServices, cli.Cfg.C2K.FlagClusterId)
			if err != nil {
				events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
				log.Fatal().Msg("Error creating service discovery and registration client")
			}
		} else if connector.ConsulDiscoveryService == cli.Cfg.SdrProvider {
			discClient, err = provider.GetConsulDiscoveryClient(cli.Cfg.HttpAddr, cli.Cfg.AsInternalServices, cli.Cfg.C2K.FlagClusterId)
			if err != nil {
				events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
				log.Fatal().Msg("Error creating service discovery and registration client")
			}
		} else if connector.NacosDiscoveryService == cli.Cfg.SdrProvider {
			discClient, err = provider.GetNacosDiscoveryClient(cli.Cfg.HttpAddr,
				cli.Cfg.Nacos.FlagUsername, cli.Cfg.Nacos.FlagPassword, cli.Cfg.Nacos.FlagNamespaceId, cli.Cfg.C2K.FlagClusterId,
				cli.Cfg.K2C.Nacos.FlagClusterId, cli.Cfg.K2C.Nacos.FlagGroupId, cli.Cfg.C2K.Nacos.FlagClusterSet, cli.Cfg.C2K.Nacos.FlagGroupSet,
				cli.Cfg.AsInternalServices)
			if err != nil {
				events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
				log.Fatal().Msg("Error creating service discovery and registration client")
			}
		} else if connector.MachineDiscoveryService == cli.Cfg.SdrProvider {
			discClient, err = provider.GetMachineDiscoveryClient(machineClient, cli.Cfg.DeriveNamespace, cli.Cfg.AsInternalServices, cli.Cfg.C2K.FlagClusterId)
			if err != nil {
				events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
				log.Fatal().Msg("Error creating service discovery and registration client")
			}
		} else {
			log.Fatal().Msg("Unsupported service discovery and registration provider")
		}

		if cli.Cfg.SyncCloudToK8s {
			go cli.SyncCtoK(ctx, kubeClient, discClient)
		}

		if cli.Cfg.SyncK8sToCloud {
			go cli.SyncKtoC(ctx, kubeClient, discClient)
		}
	}

	if cli.Cfg.SyncK8sToGateway {
		gatewayClient := gwapi.NewForConfigOrDie(kubeConfig)
		go cli.SyncKtoG(ctx, kubeClient, gatewayClient)
	}

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
	go connector.WatchMeshConfigUpdated(msgBroker, stop)

	<-stop
	log.Info().Msgf("Stopping fsm-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}
