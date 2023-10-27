// Package main implements the main entrypoint for fsm-consul-connector and utility routines to
// bootstrap the various internal components of fsm-consul-connector.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/hashicorp/consul/command/flags"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	consulConnector "github.com/flomesh-io/fsm/pkg/connector/consul"

	"github.com/flomesh-io/fsm/pkg/connector"
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
	verbosity         string
	meshName          string // An ID that uniquely identifies an FSM instance
	kubeConfigFile    string
	fsmNamespace      string
	fsmMeshConfigName string
	fsmVersion        string
	trustDomain       string
	passingOnly       bool
	filterTag         string
	prefixTag         string
	suffixTag         string
	deriveNamespace   string
	withGatewayAPI    bool

	scheme = runtime.NewScheme()
)

var (
	cliFlags  = flag.NewFlagSet("", flag.ContinueOnError)
	httpFlags = flags.HTTPFlags{}
	log       = logger.New("fsm-consul-connector")
)

func init() {
	cliFlags.StringVar(&verbosity, "verbosity", "info", "Set log verbosity level")
	cliFlags.StringVar(&meshName, "mesh-name", "", "FSM mesh name")
	cliFlags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	cliFlags.StringVar(&fsmNamespace, "fsm-namespace", "", "Namespace to which FSM belongs to.")
	cliFlags.StringVar(&fsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	cliFlags.StringVar(&fsmVersion, "fsm-version", "", "Version of FSM")

	// TODO (#4502): Remove when we add full MRC support
	cliFlags.StringVar(&trustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")
	cliFlags.StringVar(&filterTag, "filter-tag", "", "filter tag")
	cliFlags.StringVar(&prefixTag, "prefix-tag", "", "prefix tag")
	cliFlags.StringVar(&suffixTag, "suffix-tag", "", "suffix tag")
	cliFlags.BoolVar(&passingOnly, "passing-only", true, "passing only")
	cliFlags.BoolVar(&withGatewayAPI, "with-gateway-api", false, "with gateway api")
	cliFlags.StringVar(&deriveNamespace, "derive-namespace", "", "derive namespace")
	flags.Merge(cliFlags, httpFlags.ClientFlags())

	_ = clientgoscheme.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting fsm-consul-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", kubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	gatewayClient := gwapi.NewForConfigOrDie(kubeConfig)

	k8s.SetTrustDomain(trustDomain)

	connector.EnabledGatewayAPI(withGatewayAPI)
	connector.SetSyncCloudNamespace(deriveNamespace)

	// Initialize the generic Kubernetes event recorder and associate it with the fsm-consul-connector pod resource
	connectorPod, err := getConnectorPod(kubeClient)
	if err != nil {
		log.Fatal().Msg("Error fetching fsm-consul-connector pod")
	}
	eventRecorder := events.GenericEventRecorder()
	if err = eventRecorder.Initialize(connectorPod, kubeClient, fsmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err = validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := signals.RegisterExitHandlers(cancel)

	msgBroker := messaging.NewBroker(stop)

	consulClient, err := httpFlags.APIClient()
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating consul client")
	}

	sink := connector.NewSink(ctx, kubeClient, gatewayClient, fsmNamespace)
	source := &consulConnector.Source{
		ConsulClient: consulClient,
		Domain:       trustDomain,
		Sink:         sink,
		Prefix:       "",
		FilterTag:    filterTag,
		PrefixTag:    prefixTag,
		SuffixTag:    suffixTag,
		PassingOnly:  passingOnly,
	}
	sink.MicroAggregator = source
	go source.Run(ctx)

	// Build the controller and start it
	ctl := &connector.Controller{
		Resource: sink,
	}
	go ctl.Run(ctx.Done())

	version.SetMetric()
	/*
	 * Initialize fsm-consul-connector's HTTP server
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
	log.Info().Msgf("Stopping fsm-consul-connector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func parseFlags() error {
	if err := cliFlags.Parse(os.Args[1:]); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// getConnectorPod returns the fsm-consul-connector pod spec.
// The pod name is inferred from the 'CONNECTOR_POD_NAME' env variable which is set during deployment.
func getConnectorPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podName := os.Getenv("CONNECTOR_POD_NAME")
	if podName == "" {
		return nil, fmt.Errorf("CONNECTOR_POD_NAME env variable cannot be empty")
	}

	pod, err := kubeClient.CoreV1().Pods(fsmNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingConsulConnectorPod)).
			Msgf("Error retrieving fsm-consul-connector pod %s", podName)
		return nil, err
	}

	return pod, nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if meshName == "" {
		return fmt.Errorf("please specify the mesh name using --mesh-name")
	}

	if fsmNamespace == "" {
		return fmt.Errorf("please specify the FSM namespace using -fsm-namespace")
	}

	if deriveNamespace == "" {
		return fmt.Errorf("please specify the cloud derive namespace using -derive-namespace")
	}

	return nil
}
