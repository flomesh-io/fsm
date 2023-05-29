// Package main implements fsm interceptor.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/cilium/ebpf/rlimit"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/flomesh-io/fsm/pkg/cni/config"
	"github.com/flomesh-io/fsm/pkg/cni/controller/cniserver"
	"github.com/flomesh-io/fsm/pkg/cni/controller/helpers"
	"github.com/flomesh-io/fsm/pkg/cni/controller/podwatcher"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/logger"
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

	scheme = runtime.NewScheme()

	flags = pflag.NewFlagSet(`fsm-interceptor`, pflag.ExitOnError)
	log   = logger.New("fsm-interceptor/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&fsmNamespace, "fsm-namespace", "", "Namespace to which FSM belongs to.")
	flags.StringVar(&fsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	flags.StringVar(&fsmVersion, "fsm-version", "", "Version of FSM")
	flags.StringVar(&trustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")

	// Get some flags from commands
	flags.BoolVarP(&config.KernelTracing, "kernel-tracing", "d", false, "KernelTracing mode")
	flags.BoolVarP(&config.IsKind, "kind", "k", false, "Enable when Kubernetes is running in Kind")
	flags.BoolVar(&config.EnableCNI, "cni-mode", false, "Enable CNI plugin")
	flags.StringVar(&config.HostProc, "host-proc", "/host/proc", "/proc mount path")
	flags.StringVar(&config.CNIBinDir, "cni-bin-dir", "/host/opt/cni/bin", "/opt/cni/bin mount path")
	flags.StringVar(&config.CNIConfigDir, "cni-config-dir", "/host/etc/cni/net.d", "/etc/cni/net.d mount path")
	flags.StringVar(&config.HostVarRun, "host-var-run", "/host/var/run", "/var/run mount path")

	_ = clientgoscheme.AddToScheme(scheme)
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
	log.Info().Msgf("Starting fsm-interceptor %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
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

	k8s.SetTrustDomain(trustDomain)

	if err = helpers.LoadProgs(config.EnableCNI, config.KernelTracing); err != nil {
		log.Fatal().Msgf("failed to load ebpf programs: %v", err)
	}

	if err = rlimit.RemoveMemlock(); err != nil {
		log.Fatal().Msgf("remove memlock error: %v", err)
	}

	stop := make(chan struct{}, 1)
	if config.EnableCNI {
		cniReady := make(chan struct{}, 1)
		s := cniserver.NewServer(path.Join("/host", config.CNISock), "/sys/fs/bpf", cniReady, stop)
		if err = s.Start(); err != nil {
			log.Fatal().Err(err)
		}
	}
	if err = podwatcher.Run(kubeClient, stop); err != nil {
		log.Fatal().Err(err)
	}

	log.Info().Msgf("Stopping fsm-interceptor %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}
