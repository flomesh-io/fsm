package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	mapset "github.com/deckarep/golang-set"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	Cfg   = Config{}
	flags = flag.NewFlagSet("", flag.ContinueOnError)
	log   = logger.New("connector-cli")
)

// AppendSliceValue implements the flag.Value interface and allows multiple
// calls to the same variable to append a list.
type AppendSliceValue []string

func (s *AppendSliceValue) String() string {
	return strings.Join(*s, ",")
}

func (s *AppendSliceValue) Set(value string) error {
	if *s == nil {
		*s = make([]string, 0, 1)
	}
	value = strings.TrimSpace(value)
	if len(value) > 0 {
		*s = append(*s, value)
	}
	return nil
}

// ToSet creates a set from s.
func ToSet(s []string) mapset.Set {
	set := mapset.NewSet()
	for _, allow := range s {
		allow = strings.TrimSpace(allow)
		if len(allow) > 0 {
			set.Add(allow)
		}
	}
	return set
}

// ToMetaSet creates a set from s.
func ToMetaSet(s []ctv1.Metadata) mapset.Set {
	set := mapset.NewSet()
	for _, item := range s {
		set.Add(item)
	}
	return set
}

// Config is used to configure the creation of a client
type Config struct {
	Verbosity         string
	MeshName          string // An ID that uniquely identifies an FSM instance
	KubeConfigFile    string
	FsmNamespace      string
	FsmMeshConfigName string
	FsmVersion        string
	TrustDomain       string
	Workers           uint
	Limit             uint
	Burst             uint
	Timeout           uint
	SdrProvider       string
	SdrConnector      string
	LeaderElection    bool
}

func init() {
	flags.StringVar(&Cfg.Verbosity, "verbosity", "info", "Set log verbosity level")
	flags.StringVar(&Cfg.MeshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&Cfg.KubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&Cfg.FsmNamespace, "fsm-namespace", "", "Namespace to which FSM belongs to.")
	flags.StringVar(&Cfg.FsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	flags.StringVar(&Cfg.FsmVersion, "fsm-version", "", "Version of FSM")
	flags.StringVar(&Cfg.TrustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")
	flags.UintVar(&Cfg.Workers, "ctok-workers", 75, "Number of CtoK workers")
	flags.UintVar(&Cfg.Limit, "k8s-client-limit", 1000, "k8s request limit")
	flags.UintVar(&Cfg.Burst, "k8s-client-burst", 1500, "k8s request burst")
	flags.UintVar(&Cfg.Timeout, "k8s-client-timeout", 15, "k8s request timeout")
	flags.StringVar(&Cfg.SdrProvider, "sdr-provider", "", "service discovery and registration (consul, eureka, nacos, machine, gateway)")
	flags.StringVar(&Cfg.SdrConnector, "sdr-connector", "", "connector name")
	flags.BoolVar(&Cfg.LeaderElection, "leader-election", true, "leader election mode")
}

// ValidateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func ValidateCLIParams() error {
	if len(Cfg.MeshName) == 0 {
		return fmt.Errorf("please specify the mesh name using --mesh-name")
	}

	if len(Cfg.FsmNamespace) == 0 {
		return fmt.Errorf("please specify the FSM namespace using -fsm-namespace")
	}

	if len(Cfg.SdrProvider) == 0 {
		return fmt.Errorf("please specify the connector using -sdr-provider(consul/eureka/nacos/machine/gateway)")
	}

	if string(ctv1.EurekaDiscoveryService) != Cfg.SdrProvider &&
		string(ctv1.ConsulDiscoveryService) != Cfg.SdrProvider &&
		string(ctv1.NacosDiscoveryService) != Cfg.SdrProvider &&
		string(ctv1.ZookeeperDiscoveryService) != Cfg.SdrProvider &&
		string(ctv1.MachineDiscoveryService) != Cfg.SdrProvider &&
		string(ctv1.GatewayDiscoveryService) != Cfg.SdrProvider {
		return fmt.Errorf("please specify the connector using -sdr-provider(consul/eureka/nacos/machine/gateway)")
	}

	if len(Cfg.SdrConnector) == 0 {
		return fmt.Errorf("please specify the connector using -sdr-connector")
	}

	return nil
}

func ParseFlags() error {
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

func Verbosity() string {
	return Cfg.Verbosity
}
