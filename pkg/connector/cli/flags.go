package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"

	connectorv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	Cfg   = Config{Via: connector.ViaGateway}
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

type NacosCfg struct {
	FlagUsername    string
	FlagPassword    string
	FlagNamespaceId string
}

type Nacos2kCfg struct {
	FlagClusterSet []string
	FlagGroupSet   []string
}

type C2KCfg struct {
	FlagClusterId   string
	FlagPassingOnly bool
	FlagFilterTag   string
	FlagPrefixTag   string
	FlagSuffixTag   string

	Nacos Nacos2kCfg

	FlagWithGateway struct {
		Enable bool
	}
}

type K2ConsulCfg struct {
	FlagConsulNodeName string
	FlagConsulK8STag   string

	// Flags to support namespaces
	FlagConsulEnableNamespaces        bool   // Use namespacing on all components
	FlagConsulDestinationNamespace    string // Consul namespace to register everything if not mirroring
	FlagConsulEnableK8SNSMirroring    bool   // Enables mirroring of k8s namespaces into Consul
	FlagConsulK8SNSMirroringPrefix    string // Prefix added to Consul namespaces created when mirroring
	FlagConsulCrossNamespaceACLPolicy string // The name of the ACL policy to add to every created namespace if ACLs are enabled
}

type K2NacosCfg struct {
	FlagClusterId string
	FlagGroupId   string
}

type K2CCfg struct {
	FlagDefaultSync bool
	FlagSyncPeriod  time.Duration

	FlagSyncClusterIPServices     bool
	FlagSyncLoadBalancerEndpoints bool
	FlagNodePortSyncType          string

	// Flags to support Kubernetes Ingress resources
	FlagSyncIngress                bool // Register services using the hostname from an ingress resource
	FlagSyncIngressLoadBalancerIPs bool // Use the load balancer IP of an ingress resource instead of the hostname

	FlagAddServicePrefix               string
	FlagAddK8SNamespaceAsServiceSuffix bool
	FlagAppendTags                     []string
	FlagAppendMetadataKeys             []string
	FlagAppendMetadataValues           []string
	FlagAllowK8SNamespaces             []string // K8s namespaces to explicitly inject
	FlagDenyK8SNamespaces              []string // K8s namespaces to deny injection (has precedence)

	Consul K2ConsulCfg
	Nacos  K2NacosCfg

	FlagWithGateway struct {
		Enable bool
	}
}

type K2GCfg struct {
	FlagDefaultSync        bool
	FlagSyncPeriod         time.Duration
	FlagAllowK8SNamespaces []string // K8s namespaces to explicitly inject
	FlagDenyK8SNamespaces  []string // K8s namespaces to deny injection (has precedence)
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
	SdrProvider       string
	SdrConnector      string

	DeriveNamespace    string
	AsInternalServices bool

	HttpAddr         string
	SyncCloudToK8s   bool
	SyncK8sToCloud   bool
	SyncK8sToGateway bool

	Nacos NacosCfg

	C2K C2KCfg
	K2C K2CCfg
	K2G K2GCfg

	Via *connector.Gateway
}

func init() {
	flags.StringVar(&Cfg.Verbosity, "verbosity", "info", "Set log verbosity level")
	flags.StringVar(&Cfg.MeshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&Cfg.KubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&Cfg.FsmNamespace, "fsm-namespace", "", "Namespace to which FSM belongs to.")
	flags.StringVar(&Cfg.FsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	flags.StringVar(&Cfg.FsmVersion, "fsm-version", "", "Version of FSM")
	flags.StringVar(&Cfg.TrustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")

	flags.StringVar(&Cfg.SdrProvider, "sdr-provider", "", "service discovery and registration (consul, eureka, nacos, machine, gateway)")
	flags.StringVar(&Cfg.SdrConnector, "sdr-connector", "", "connector name")
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

	if string(connectorv1alpha1.EurekaDiscoveryService) != Cfg.SdrProvider &&
		string(connectorv1alpha1.ConsulDiscoveryService) != Cfg.SdrProvider &&
		string(connectorv1alpha1.NacosDiscoveryService) != Cfg.SdrProvider &&
		string(connectorv1alpha1.MachineDiscoveryService) != Cfg.SdrProvider &&
		string(connectorv1alpha1.GatewayDiscoveryService) != Cfg.SdrProvider {
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
