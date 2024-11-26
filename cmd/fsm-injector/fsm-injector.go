// Package main implements the main entrypoint for fsm-injector and utility routines to
// bootstrap the various internal components of fsm-injector.
// fsm-injector provides the automatic sidecar injection capability in FSM.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	smiAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	pluginClientset "github.com/flomesh-io/fsm/pkg/gen/client/plugin/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/service"
	sidecarv1 "github.com/flomesh-io/fsm/pkg/sidecar/v1"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	policyClientset "github.com/flomesh-io/fsm/pkg/gen/client/policy/clientset/versioned"

	_ "github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/driver"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/certificate/providers"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/httpserver"
	"github.com/flomesh-io/fsm/pkg/injector"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
	"github.com/flomesh-io/fsm/pkg/reconciler"
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/version"
)

var (
	verbosity          string
	meshName           string // An ID that uniquely identifies an FSM instance
	kubeConfigFile     string
	fsmNamespace       string
	webhookConfigName  string
	caBundleSecretName string
	fsmMeshConfigName  string
	webhookTimeout     int32
	fsmVersion         string
	trustDomain        string

	certProviderKind          string
	enableMeshRootCertificate bool

	enableReconciler bool

	tresorOptions      providers.TresorOptions
	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

	fsmContainerPullPolicy string

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`fsm-injector`, pflag.ExitOnError)
	log   = logger.New("fsm-injector/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&fsmNamespace, "fsm-namespace", "", "Namespace to which FSM belongs to.")
	flags.StringVar(&webhookConfigName, "webhook-config-name", "", "Name of the MutatingWebhookConfiguration to be configured by fsm-injector")
	flags.Int32Var(&webhookTimeout, "webhook-timeout", int32(20), "Timeout of the MutatingWebhookConfiguration")
	flags.StringVar(&fsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	flags.StringVar(&fsmVersion, "fsm-version", "", "Version of FSM")

	// Generic certificate manager/provider options
	flags.StringVar(&certProviderKind, "certificate-manager", providers.TresorKind.String(), fmt.Sprintf("Certificate manager, one of [%v]", providers.ValidCertificateProviders))
	flags.BoolVar(&enableMeshRootCertificate, "enable-mesh-root-certificate", false, "Enable unsupported MeshRootCertificate to create the FSM Certificate Manager")
	flags.StringVar(&caBundleSecretName, "ca-bundle-secret-name", "", "Name of the Kubernetes Secret for the FSM CA bundle")

	// TODO (#4502): Remove when we add full MRC support
	flags.StringVar(&trustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")

	// Vault certificate manager/provider options
	flags.StringVar(&vaultOptions.VaultProtocol, "vault-protocol", "http", "Host name of the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultHost, "vault-host", "vault.default.svc.cluster.local", "Host name of the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultToken, "vault-token", "", "Secret token for the the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultRole, "vault-role", "flomesh", "Name of the Vault role dedicated to Flomesh Service Mesh")
	flags.IntVar(&vaultOptions.VaultPort, "vault-port", 8200, "Port of the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultTokenSecretName, "vault-token-secret-name", "", "Name of the secret storing the Vault token used in FSM")
	flags.StringVar(&vaultOptions.VaultTokenSecretKey, "vault-token-secret-key", "", "Key for the vault token used in FSM")

	// Cert-manager certificate manager/provider options
	flags.StringVar(&certManagerOptions.IssuerName, "cert-manager-issuer-name", "fsm-ca", "cert-manager issuer name")
	flags.StringVar(&certManagerOptions.IssuerKind, "cert-manager-issuer-kind", "Issuer", "cert-manager issuer kind")
	flags.StringVar(&certManagerOptions.IssuerGroup, "cert-manager-issuer-group", "cert-manager.io", "cert-manager issuer group")

	// Reconciler options
	flags.BoolVar(&enableReconciler, "enable-reconciler", false, "Enable reconciler for CDRs, mutating webhook and validating webhook")

	flags.StringVar(&fsmContainerPullPolicy, "fsm-container-pull-policy", "", "The pullPolicy to use for injected init and healthcheck containers")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
}

// TODO(#4502): This function can be deleted once we get rid of cert options.
func getCertOptions() (providers.Options, error) {
	switch providers.Kind(certProviderKind) {
	case providers.TresorKind:
		tresorOptions.SecretName = caBundleSecretName
		return tresorOptions, nil
	case providers.VaultKind:
		vaultOptions.VaultTokenSecretNamespace = fsmNamespace
		return vaultOptions, nil
	case providers.CertManagerKind:
		return certManagerOptions, nil
	}
	return nil, fmt.Errorf("unknown certificate provider kind: %s", certProviderKind)
}

func main() {
	log.Info().Msgf("Starting fsm-injector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
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
	policyClient := policyClientset.NewForConfigOrDie(kubeConfig)
	pluginClient := pluginClientset.NewForConfigOrDie(kubeConfig)
	machineClient := machineClientset.NewForConfigOrDie(kubeConfig)
	configClient := configClientset.NewForConfigOrDie(kubeConfig)

	service.SetTrustDomain(trustDomain)

	// Initialize the generic Kubernetes event recorder and associate it with the fsm-injector pod resource
	injectorPod, err := getInjectorPod(kubeClient)
	if err != nil {
		log.Fatal().Msg("Error fetching fsm-injector pod")
	}
	eventRecorder := events.GenericEventRecorder()
	if err := eventRecorder.Initialize(injectorPod, kubeClient, fsmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := signals.RegisterExitHandlers(cancel)

	// Start the default metrics store
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.CertIssuedCount,
		metricsstore.DefaultMetricsStore.CertIssuedTime,
		metricsstore.DefaultMetricsStore.ErrCodeCounter,
		metricsstore.DefaultMetricsStore.HTTPResponseTotal,
		metricsstore.DefaultMetricsStore.HTTPResponseDuration,
		metricsstore.DefaultMetricsStore.AdmissionWebhookResponseTotal,
		metricsstore.DefaultMetricsStore.ReconciliationTotal,
	)

	msgBroker := messaging.NewBroker(stop)

	smiTrafficSplitClientSet := smiTrafficSplitClient.NewForConfigOrDie(kubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClient.NewForConfigOrDie(kubeConfig)
	smiTrafficTargetClientSet := smiAccessClient.NewForConfigOrDie(kubeConfig)

	informerCollection, err := informers.NewInformerCollection(meshName, stop,
		informers.WithKubeClient(kubeClient),
		informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
		informers.WithConfigClient(configClient, fsmMeshConfigName, fsmNamespace),
		informers.WithPolicyClient(policyClient),
		informers.WithMachineClient(machineClient),
	)

	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating informer collection")
	}

	// Initialize Configurator to watch resources in the config.flomesh.io API group
	cfg := configurator.NewConfigurator(informerCollection, fsmNamespace, fsmMeshConfigName, msgBroker)
	err = sidecarv1.InstallDriver(cfg.GetSidecarClass())
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating sidecar driver")
	}

	// Initialize kubernetes.Controller to watch kubernetes resources
	kubeController := k8s.NewKubernetesController(informerCollection, policyClient, pluginClient, msgBroker, k8s.Namespaces)

	certOpts, err := getCertOptions()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting certificate options")
	}
	// Intitialize certificate manager/provider
	var certManager *certificate.Manager
	if enableMeshRootCertificate {
		certManager, err = providers.NewCertificateManagerFromMRC(ctx, kubeClient, kubeConfig, cfg, fsmNamespace,
			certOpts, msgBroker, informerCollection, 5*time.Second)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
				"Error initializing certificate manager of kind %s from MRC", certProviderKind)
		}
	} else {
		certManager, err = providers.NewCertificateManager(ctx, kubeClient, kubeConfig, cfg, fsmNamespace,
			certOpts, msgBroker, 5*time.Second, trustDomain)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
				"Error initializing certificate manager of kind %s", certProviderKind)
		}
	}

	// Initialize the sidecar injector webhook
	if err := injector.NewMutatingWebhook(ctx, kubeClient, certManager, kubeController, meshName, fsmNamespace, webhookConfigName, fsmVersion, webhookTimeout, enableReconciler, cfg, corev1.PullPolicy(fsmContainerPullPolicy)); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating sidecar injector webhook")
	}

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

	if enableReconciler {
		log.Info().Msgf("FSM reconciler enabled for sidecar injector webhook")
		err = reconciler.NewReconcilerClient(kubeClient, nil, meshName, fsmVersion, stop, reconciler.MutatingWebhookInformerKey)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating reconciler client to reconcile sidecar injector webhook")
		}
	}

	<-stop
	log.Info().Msgf("Stopping fsm-injector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// getInjectorPod returns the fsm-injector pod spec.
// The pod name is inferred from the 'INJECTOR_POD_NAME' env variable which is set during deployment.
func getInjectorPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podName := os.Getenv("INJECTOR_POD_NAME")
	if podName == "" {
		return nil, fmt.Errorf("INJECTOR_POD_NAME env variable cannot be empty")
	}

	pod, err := kubeClient.CoreV1().Pods(fsmNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingInjectorPod)).
			Msgf("Error retrieving fsm-injector pod %s", podName)
		return nil, err
	}

	return pod, nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if meshName == "" {
		return fmt.Errorf("Please specify the mesh name using --mesh-name")
	}

	if fsmNamespace == "" {
		return fmt.Errorf("Please specify the FSM namespace using --fsm-namespace")
	}

	if webhookConfigName == "" {
		return fmt.Errorf("Please specify the mutatingwebhookconfiguration name using --webhook-config-name value")
	}

	if caBundleSecretName == "" {
		return fmt.Errorf("Please specify the CA bundle secret name using --ca-bundle-secret-name")
	}

	return nil
}
