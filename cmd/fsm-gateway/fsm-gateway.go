/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package main implements the main entry point for the Flomesh Gateway.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gwscheme "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/scheme"

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/version"
)

type metadata struct {
	PodName      string `envconfig:"POD_NAME" required:"true" split_words:"true"`
	PodNamespace string `envconfig:"POD_NAMESPACE" required:"true" split_words:"true"`
}

var (
	flags  = pflag.NewFlagSet(`fsm-gateway`, pflag.ExitOnError)
	log    = logger.New("fsm-gateway/main")
	scheme = runtime.NewScheme()
)

var (
	verbosity         string
	meshName          string // An ID that uniquely identifies an FSM instance
	fsmNamespace      string
	fsmMeshConfigName string
	fsmVersion        string
	gatewayNamespace  string
	gatewayName       string
	serviceName       string

	meta metadata
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", constants.DefaultFSMLogLevel, "Set boot log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&fsmNamespace, "fsm-namespace", "", "FSM controller's namespace")
	flags.StringVar(&fsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	flags.StringVar(&fsmVersion, "fsm-version", "", "Version of FSM")
	flags.StringVar(&gatewayNamespace, "gateway-namespace", "", "Namespace of Gateway")
	flags.StringVar(&gatewayName, "gateway-name", "", "Name of Gateway")
	flags.StringVar(&serviceName, "service-name", "", "Name of Gateway Service")

	meta = getMetadata()

	_ = clientgoscheme.AddToScheme(scheme)
	_ = gwscheme.AddToScheme(scheme)
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
	log.Info().Msgf("Starting fsm-gateway %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
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
		log.Error().Msgf("Error creating informer collection: %s", err)
	}

	cfg := configurator.NewConfigurator(informerCollection, fsmNamespace, fsmMeshConfigName, msgBroker)

	if !cfg.IsGatewayAPIEnabled() {
		log.Error().Msgf("GatewayAPI is not enabled, FSM doesn't support Ingress and GatewayAPI are both enabled.")
		os.Exit(1)
	}

	// codebase URL
	url := codebase(cfg)
	log.Info().Msgf("Gateway Repo = %q", url)

	// calculate pipy spawn
	spawn := calcPipySpawn(kubeClient)
	log.Info().Msgf("PIPY SPAWN = %d", spawn)

	startHTTPServer()

	startPipy(spawn, url, cfg)

	<-stop
	cancel()
	log.Info().Msgf("Stopping fsm-gateway %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func codebase(cfg configurator.Configurator) string {
	repoHost := fmt.Sprintf("%s.%s.svc", constants.FSMControllerName, fsmNamespace)
	repoPort := cfg.GetProxyServerPort()
	return fmt.Sprintf("%s://%s:%d/repo%s/", "http", repoHost, repoPort, utils.GatewayCodebasePath(gatewayNamespace, gatewayName))
}

func calcPipySpawn(kubeClient kubernetes.Interface) int64 {
	cpuLimits, err := getGatewayCPULimitsQuota(kubeClient)
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

func getGatewayPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podNamespace := meta.PodNamespace
	podName := meta.PodName

	pod, err := kubeClient.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error().Msgf("Error retrieving gateway pod %s", podName)
		return nil, err
	}

	return pod, nil
}

func getGatewayCPULimitsQuota(kubeClient kubernetes.Interface) (*resource.Quantity, error) {
	pod, err := getGatewayPod(kubeClient)
	if err != nil {
		return nil, err
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == "gateway" {
			return c.Resources.Limits.Cpu(), nil
		}
	}

	return nil, fmt.Errorf("no container named 'gateway' in POD %q", pod.Name)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

func startPipy(spawn int64, url string, _ configurator.Configurator) {
	args := []string{url}
	if spawn > 1 {
		args = append([]string{"--reuse-port", fmt.Sprintf("--threads=%d", spawn)}, args...)
	}

	if verbosity != "disabled" {
		args = append([]string{fmt.Sprintf("--log-level=%s", utils.PipyLogLevelByVerbosity(verbosity))}, args...)
	}

	args = append(args, fmt.Sprintf("--admin-port=%s:%d", constants.LocalhostIPAddress, constants.FSMGatewayAdminPort))

	// arguments for FGW
	args = append(args, "--args", "--watch")

	if verbosity == "debug" || verbosity == "trace" {
		args = append(args, "--debug")
	}

	cmd := exec.Command("pipy", args...) // #nosec G204
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info().Msgf("cmd = %v", cmd)

	if err := cmd.Run(); err != nil {
		log.Fatal().Err(err)
		os.Exit(1)
	}

	if cmd.ProcessState != nil {
		log.Info().Msgf("PIPY process state: %v", cmd.ProcessState.String())
	}

	if cmd.Process != nil {
		// detach it from the go process
		if err := cmd.Process.Release(); err != nil {
			log.Fatal().Err(err)
		}
	}
}

func startHTTPServer() {
	// Initialize FSM's http service server
	httpServer := httpserver.NewHTTPServer(constants.FSMGatewayHTTPServerPort)

	// Metrics
	httpServer.AddHandler(constants.MetricsPath, metricsHandler())
	// Version
	httpServer.AddHandler(constants.VersionPath, version.GetVersionHandler())
	// Health Check
	httpServer.AddHandler(constants.HealthCheckPath, healthCheckHandler())

	// Start HTTP server
	err := httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start FSM Gateway metrics/probes HTTP server")
		os.Exit(1)
	}
}

func metricsHandler() http.Handler {
	handleInternalError := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("Internal Server Error")); err != nil {
			log.Error().Err(err).Msg("Failed to write response")
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestURL := fmt.Sprintf("http://%s:%d", constants.LocalhostIPAddress, constants.FSMGatewayAdminPort)

		newReq, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			log.Error().Msgf("Could not create request: %s\n", err)
			handleInternalError(w)
			return
		}

		res, err := http.DefaultClient.Do(newReq)
		if err != nil {
			log.Error().Msgf("Error making http request: %s\n", err)
			handleInternalError(w)
			return
		}

		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Error().Msgf("Could not read response body: %s\n", err)
			handleInternalError(w)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, string(resBody))
	})
}

func healthCheckHandler() http.Handler {
	setHealthcheckResponse := func(w http.ResponseWriter, responseCode int, msg string) {
		w.WriteHeader(responseCode)
		if _, err := w.Write([]byte(msg)); err != nil {
			log.Error().Err(err).Msg("Failed to write response")
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		address := fmt.Sprintf("%s:%d", constants.LocalhostIPAddress, constants.FSMGatewayAdminPort)

		conn, err := net.Dial("tcp", address)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to establish connection to %s", address)
			setHealthcheckResponse(w, http.StatusNotFound, "FAILED")
			return
		}

		if err = conn.Close(); err != nil {
			log.Error().Err(err).Msgf("Failed to close connection to %s", address)
		}

		setHealthcheckResponse(w, http.StatusOK, "OK")
	})
}
