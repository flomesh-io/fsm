// Package main implements the main entrypoint for fsm-controller and utility routines to
// bootstrap the various internal components of fsm-controller.
// fsm-controller is the core control plane component in FSM responsible for programming sidecar proxies.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-logr/zerologr"

	"github.com/flomesh-io/fsm/pkg/dns"
	connectorClientset "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	policyAttachmentClientset "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"
	mgrecon "github.com/flomesh-io/fsm/pkg/manager/reconciler"
	sidecarv1 "github.com/flomesh-io/fsm/pkg/sidecar/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	ctrlwh "sigs.k8s.io/controller-runtime/pkg/webhook"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/manager/basic"
	"github.com/flomesh-io/fsm/pkg/manager/listeners"
	"github.com/flomesh-io/fsm/pkg/manager/logging"
	mrepo "github.com/flomesh-io/fsm/pkg/manager/repo"
	"github.com/flomesh-io/fsm/pkg/repo"

	smiAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gwscheme "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/scheme"

	connectorscheme "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned/scheme"
	extscheme "github.com/flomesh-io/fsm/pkg/gen/client/extension/clientset/versioned/scheme"
	machinescheme "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned/scheme"
	mcscheme "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/clientset/versioned/scheme"
	nsigscheme "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned/scheme"
	pascheme "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned/scheme"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	multiclusterClientset "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/clientset/versioned"
	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"
	networkingClientset "github.com/flomesh-io/fsm/pkg/gen/client/networking/clientset/versioned"
	pluginClientset "github.com/flomesh-io/fsm/pkg/gen/client/plugin/clientset/versioned"
	policyClientset "github.com/flomesh-io/fsm/pkg/gen/client/policy/clientset/versioned"

	_ "github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/driver"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/certificate/providers"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/httpserver"
	"github.com/flomesh-io/fsm/pkg/ingress"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
	"github.com/flomesh-io/fsm/pkg/multicluster"
	"github.com/flomesh-io/fsm/pkg/plugin"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/providers/fsm"
	"github.com/flomesh-io/fsm/pkg/providers/kube"
	"github.com/flomesh-io/fsm/pkg/reconciler"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/smi"
	"github.com/flomesh-io/fsm/pkg/validator"
	"github.com/flomesh-io/fsm/pkg/version"
)

const (
	xdsServerCertificateCommonName = "ads"
)

var (
	verbosity                  string
	meshName                   string // An ID that uniquely identifies an FSM instance
	fsmNamespace               string
	fsmServiceAccount          string
	validatorWebhookConfigName string
	caBundleSecretName         string
	fsmMeshConfigName          string
	fsmVersion                 string
	trustDomain                string

	certProviderKind          string
	enableMeshRootCertificate bool

	tresorOptions      providers.TresorOptions
	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

	enableReconciler      bool
	enableMultiClusters   bool
	validateTrafficTarget bool

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`fsm-controller`, pflag.ExitOnError)
	log   = logger.New("fsm-controller/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", constants.DefaultFSMLogLevel, "Set boot log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&fsmNamespace, "fsm-namespace", "", "FSM controller's namespace")
	flags.StringVar(&fsmServiceAccount, "fsm-service-account", "", "FSM controller's service account")
	flags.StringVar(&validatorWebhookConfigName, "validator-webhook-config", "", "Name of the ValidatingWebhookConfiguration for the resource validator webhook")
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
	flags.BoolVar(&enableMultiClusters, "enable-multi-clusters", false, "Enable multi-clusters")
	flags.BoolVar(&validateTrafficTarget, "validate-traffic-target", true, "Enable traffic target validation")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
	_ = gwscheme.AddToScheme(scheme)
	_ = mcscheme.AddToScheme(scheme)
	_ = nsigscheme.AddToScheme(scheme)
	_ = pascheme.AddToScheme(scheme)
	_ = machinescheme.AddToScheme(scheme)
	_ = connectorscheme.AddToScheme(scheme)
	_ = extscheme.AddToScheme(scheme)
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

//gocyclo:ignore
func main() {
	log.Info().Msgf("Starting fsm-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Str(errcode.Kind, errcode.ErrInvalidCLIArgument.String()).Msg("Error parsing cmd line arguments")
	}

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating kube configs using in-cluster config")
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	policyClient := policyClientset.NewForConfigOrDie(kubeConfig)
	pluginClient := pluginClientset.NewForConfigOrDie(kubeConfig)
	machineClient := machineClientset.NewForConfigOrDie(kubeConfig)
	connectorClient := connectorClientset.NewForConfigOrDie(kubeConfig)
	configClient := configClientset.NewForConfigOrDie(kubeConfig)
	multiclusterClient := multiclusterClientset.NewForConfigOrDie(kubeConfig)
	networkingClient := networkingClientset.NewForConfigOrDie(kubeConfig)

	service.SetTrustDomain(trustDomain)

	// Initialize the generic Kubernetes event recorder and associate it with the fsm-controller pod resource
	controllerPod, err := getFSMControllerPod(kubeClient)
	if err != nil {
		log.Fatal().Msg("Error fetching fsm-controller pod")
	}
	eventRecorder := events.GenericEventRecorder()
	if err := eventRecorder.Initialize(controllerPod, kubeClient, fsmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	background := fctx.ControllerContext{
		FsmNamespace: fsmNamespace,
		KubeConfig:   kubeConfig,
	}
	ctx, cancel := context.WithCancel(&background)
	stop := signals.RegisterExitHandlers(cancel)
	background.CancelFunc = cancel
	background.Stop = stop

	// Start the default metrics store
	startMetricsStore()

	msgBroker := messaging.NewBroker(stop)

	smiTrafficSplitClientSet := smiTrafficSplitClient.NewForConfigOrDie(kubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClient.NewForConfigOrDie(kubeConfig)
	smiTrafficTargetClientSet := smiAccessClient.NewForConfigOrDie(kubeConfig)
	gatewayAPIClient := gatewayApiClientset.NewForConfigOrDie(kubeConfig)
	namespacedIngressClient := nsigClientset.NewForConfigOrDie(kubeConfig)
	policyAttachmentClient := policyAttachmentClientset.NewForConfigOrDie(kubeConfig)

	opts := []informers.InformerCollectionOption{
		informers.WithKubeClient(kubeClient),
		informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
		informers.WithConfigClient(configClient, fsmMeshConfigName, fsmNamespace),
		informers.WithPolicyClient(policyClient),
		informers.WithPluginClient(pluginClient),
		informers.WithMachineClient(machineClient),
		informers.WithConnectorClient(connectorClient),
		informers.WithNetworkingClient(networkingClient),
		informers.WithIngressClient(kubeClient, namespacedIngressClient),
		informers.WithGatewayAPIClient(gatewayAPIClient),
		informers.WithPolicyAttachmentClientV2(gatewayAPIClient, policyAttachmentClient),
	}

	if enableMultiClusters {
		opts = append(opts, informers.WithMultiClusterClient(multiclusterClient))
	}

	informerCollection, err := informers.NewInformerCollection(meshName, stop, opts...)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating informer collection")
	}

	// This component will be watching resources in the config.flomesh.io API group
	cfg := configurator.NewConfigurator(informerCollection, fsmNamespace, fsmMeshConfigName, msgBroker)
	k8sClient := k8s.NewKubernetesController(informerCollection, policyClient, pluginClient, msgBroker)
	meshSpec := smi.NewSMIClient(informerCollection, fsmNamespace, k8sClient, msgBroker)

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
				"Error fetching certificate manager of kind %s from MRC", certProviderKind)
		}
	} else {
		certManager, err = providers.NewCertificateManager(ctx, kubeClient, kubeConfig, cfg, fsmNamespace,
			certOpts, msgBroker, 5*time.Second, trustDomain)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
				"Error fetching certificate manager of kind %s", certProviderKind)
		}
	}

	policyController := policy.NewPolicyController(informerCollection, kubeClient, k8sClient, msgBroker)
	pluginController := plugin.NewPluginController(informerCollection, kubeClient, k8sClient, msgBroker)
	multiclusterController := multicluster.NewMultiClusterController(informerCollection, kubeClient, k8sClient, msgBroker)

	kubeProvider := kube.NewClient(k8sClient, cfg)

	endpointsProviders := []endpoint.Provider{kubeProvider}
	serviceProviders := []service.Provider{kubeProvider}

	if enableMultiClusters {
		multiclusterProvider := fsm.NewClient(multiclusterController, cfg)
		endpointsProviders = append(endpointsProviders, multiclusterProvider)
		serviceProviders = append(serviceProviders, multiclusterProvider)
	}

	if err := ingress.Initialize(kubeClient, k8sClient, stop, cfg, certManager, msgBroker); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Ingress client")
	}

	meshCatalog := catalog.NewMeshCatalog(
		k8sClient,
		meshSpec,
		certManager,
		policyController,
		pluginController,
		multiclusterController,
		stop,
		cfg,
		serviceProviders,
		endpointsProviders,
		msgBroker,
	)

	background.Configurator = cfg
	background.MeshCatalog = meshCatalog
	background.CertManager = certManager
	background.MsgBroker = msgBroker

	// Health/Liveness probes
	var funcProbes []health.Probes
	if cfg.GetTrafficInterceptionMode() == constants.TrafficInterceptionModePodLevel {
		err = sidecarv1.InstallDriver(cfg.GetSidecarClass())
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating sidecar driver")
		}

		proxyServiceCert, err := certManager.IssueCertificate(xdsServerCertificateCommonName, certificate.Internal)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.CertificateIssuanceFailure, "Error issuing XDS certificate to ADS server")
		}

		background.ProxyServiceCert = proxyServiceCert
		background.ProxyServerPort = cfg.GetProxyServerPort()

		// Create and start the sidecar proxy service
		healthProbes, err := sidecarv1.Start(ctx)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error initializing proxy control server")
		}

		go dns.WatchAndUpdateLocalDNSProxy(msgBroker, stop)
		// Start the k8s pod watcher that updates corresponding k8s secrets
		go k8s.WatchAndUpdateProxyBootstrapSecret(kubeClient, msgBroker, stop)

		funcProbes = append(funcProbes, healthProbes)
	}

	dns.Init(k8sClient, cfg)

	clientset := extensionsClientset.NewForConfigOrDie(kubeConfig)

	if err = validator.NewValidatingWebhook(ctx, cfg, validatorWebhookConfigName, fsmNamespace, fsmVersion, meshName, enableReconciler, validateTrafficTarget, certManager, kubeClient, k8sClient, policyController); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error starting the validating webhook server")
	}

	funcProbes = append(funcProbes, smi.HealthChecker{DiscoveryClient: clientset.Discovery()})

	version.SetMetric()

	// Initialize FSM's http service server
	httpServer := httpserver.NewHTTPServer(constants.FSMHTTPServerPort)
	httpServer.AddHandlers(map[string]http.Handler{
		constants.FSMControllerReadinessPath: health.ReadinessHandler(funcProbes, nil),
		constants.FSMControllerLivenessPath:  health.LivenessHandler(funcProbes, nil),
	})
	// Metrics
	httpServer.AddHandler(constants.MetricsPath, metricsstore.DefaultMetricsStore.Handler())
	// Version
	httpServer.AddHandler(constants.VersionPath, version.GetVersionHandler())
	// Supported SMI Versions
	httpServer.AddHandler(constants.FSMControllerSMIVersionPath, smi.GetSmiClientVersionHTTPHandler())

	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start FSM metrics/probes HTTP server")
	}

	// Start the global log level watcher that updates the log level dynamically
	go k8s.WatchAndUpdateLogLevel(msgBroker, stop)

	if enableReconciler {
		log.Info().Msgf("FSM reconciler enabled for validating webhook")
		err = reconciler.NewReconcilerClient(kubeClient, nil, meshName, fsmVersion, stop, reconciler.ValidatingWebhookInformerKey)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating reconciler client to reconcile validating webhook")
		}
	}

	ctrl.SetLogger(zerologr.New(&log))
	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
		Scheme:                  scheme,
		LeaderElection:          true,
		LeaderElectionNamespace: cfg.GetFSMNamespace(),
		LeaderElectionID:        constants.FSMControllerLeaderElectionID,
		WebhookServer:           ctrlwh.NewServer(ctrlwh.Options{Port: constants.FSMWebhookPort}),
	})
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating manager")
	}

	background.Client = mgr.GetClient()
	background.Manager = mgr
	background.Scheme = mgr.GetScheme()
	background.KubeClient = kubeClient
	background.RepoClient = repo.NewRepoClient(fmt.Sprintf("%s://%s:%d", "http", cfg.GetRepoServerIPAddr(), cfg.GetProxyServerPort()), cfg.GetFSMLogLevel())
	background.InformerCollection = informerCollection
	background.MeshName = meshName
	background.FSMVersion = fsmVersion
	background.TrustDomain = trustDomain

	for _, f := range []func(context.Context) error{
		mrepo.InitRepo,
		basic.SetupHTTP,
		basic.SetupTLS,
		logging.SetupLogging,
		//webhook.RegisterWebHooks,
		mgrecon.RegisterControllers,
		mgrecon.RegisterWebhooksAndReconcilers,
	} {
		if err := f(ctx); err != nil {
			log.Error().Msgf("Failed to startup: %s", err)
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error setting up manager")
		}
	}

	if cfg.IsIngressEnabled() {
		go listeners.WatchAndUpdateIngressConfig(kubeClient, msgBroker, fsmNamespace, certManager, background.RepoClient, stop)
		go listeners.WatchAndUpdateLoggingConfig(kubeClient, msgBroker, background.RepoClient, stop)
	}

	if err := mgr.Start(ctx); err != nil {
		log.Fatal().Msgf("problem running manager, %s", err)
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error starting manager")
	}

	<-stop
	cancel()
	log.Info().Msgf("Stopping fsm-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

// Start the metric store, register the metrics FSM will expose
func startMetricsStore() {
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.K8sAPIEventCounter,
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter,
		metricsstore.DefaultMetricsStore.ProxyConnectCount,
		metricsstore.DefaultMetricsStore.ProxyReconnectCount,
		metricsstore.DefaultMetricsStore.ProxyConfigUpdateTime,
		metricsstore.DefaultMetricsStore.ProxyBroadcastEventCount,
		metricsstore.DefaultMetricsStore.ProxyResponseSendSuccessCount,
		metricsstore.DefaultMetricsStore.ProxyResponseSendErrorCount,
		metricsstore.DefaultMetricsStore.ErrCodeCounter,
		metricsstore.DefaultMetricsStore.HTTPResponseTotal,
		metricsstore.DefaultMetricsStore.HTTPResponseDuration,
		metricsstore.DefaultMetricsStore.FeatureFlagEnabled,
		metricsstore.DefaultMetricsStore.VersionInfo,
		metricsstore.DefaultMetricsStore.ProxyXDSRequestCount,
		metricsstore.DefaultMetricsStore.ProxyMaxConnectionsRejected,
		metricsstore.DefaultMetricsStore.AdmissionWebhookResponseTotal,
		metricsstore.DefaultMetricsStore.EventsQueued,
		metricsstore.DefaultMetricsStore.ReconciliationTotal,
		metricsstore.DefaultMetricsStore.IngressBroadcastEventCount,
		metricsstore.DefaultMetricsStore.GatewayBroadcastEventCounter,
	)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

//lint:ignore U1000 This is used in the tests
func joinURL(baseURL string, paths ...string) string {
	p := path.Join(paths...)
	return fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), strings.TrimLeft(p, "/"))
}

// getFSMControllerPod returns the fsm-controller pod.
// The pod name is inferred from the 'CONTROLLER_POD_NAME' env variable which is set during deployment.
func getFSMControllerPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podName := os.Getenv("CONTROLLER_POD_NAME")
	if podName == "" {
		return nil, fmt.Errorf("CONTROLLER_POD_NAME env variable cannot be empty")
	}

	pod, err := kubeClient.CoreV1().Pods(fsmNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingControllerPod)).
			Msgf("Error retrieving fsm-controller pod %s", podName)
		return nil, err
	}

	return pod, nil
}
