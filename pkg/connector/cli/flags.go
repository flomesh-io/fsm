package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/connector"
)

var (
	Cfg   Config
	flags = flag.NewFlagSet("", flag.ContinueOnError)
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

	*s = append(*s, value)
	return nil
}

// ToSet creates a set from s.
func ToSet(s []string) mapset.Set {
	set := mapset.NewSet()
	for _, allow := range s {
		set.Add(allow)
	}
	return set
}

type C2KCfg struct {
	FlagPassingOnly    bool
	FlagFilterTag      string
	FlagPrefixTag      string
	FlagSuffixTag      string
	FlagWithGatewayAPI bool
}

type K2CCfg struct {
	FlagConsulK8STag          string
	FlagConsulNodeName        string
	FlagK8SDefault            bool
	FlagK8SServicePrefix      string
	FlagConsulServicePrefix   string
	FlagK8SWriteNamespace     string
	FlagConsulWritePeriod     time.Duration
	FlagSyncClusterIPServices bool
	FlagSyncLBEndpoints       bool
	FlagNodePortSyncType      string
	FlagAddK8SNamespaceSuffix bool

	// Flags to support namespaces
	FlagEnableNamespaces           bool     // Use namespacing on all components
	FlagConsulDestinationNamespace string   // Consul namespace to register everything if not mirroring
	FlagAllowK8sNamespacesList     []string // K8s namespaces to explicitly inject
	FlagDenyK8sNamespacesList      []string // K8s namespaces to deny injection (has precedence)
	FlagEnableK8SNSMirroring       bool     // Enables mirroring of k8s namespaces into Consul
	FlagK8SNSMirroringPrefix       string   // Prefix added to Consul namespaces created when mirroring
	FlagCrossNamespaceACLPolicy    string   // The name of the ACL policy to add to every created namespace if ACLs are enabled

	// Flags to support Kubernetes Ingress resources
	FlagEnableIngress   bool // Register services using the hostname from an ingress resource
	FlagLoadBalancerIPs bool // Use the load balancer IP of an ingress resource instead of the hostname
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
	DeriveNamespace   string
	SdrProvider       string
	HttpAddr          string

	c2k C2KCfg
	k2c K2CCfg
}

func init() {
	flags.StringVar(&Cfg.Verbosity, "verbosity", "info", "Set log verbosity level")
	flags.StringVar(&Cfg.MeshName, "mesh-name", "", "FSM mesh name")
	flags.StringVar(&Cfg.KubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&Cfg.FsmNamespace, "fsm-namespace", "", "Namespace to which FSM belongs to.")
	flags.StringVar(&Cfg.FsmMeshConfigName, "fsm-config-name", "fsm-mesh-config", "Name of the FSM MeshConfig")
	flags.StringVar(&Cfg.FsmVersion, "fsm-version", "", "Version of FSM")
	flags.StringVar(&Cfg.TrustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")
	flags.StringVar(&Cfg.DeriveNamespace, "derive-namespace", "", "derive namespace")
	flags.StringVar(&Cfg.SdrProvider, "sdr-provider", "", "service discovery and registration (eureka, consul)")
	flags.StringVar(&Cfg.HttpAddr, "sdr-http-addr", "", "http addr")

	flags.StringVar(&Cfg.c2k.FlagFilterTag, "filter-tag", "", "filter tag")
	flags.StringVar(&Cfg.c2k.FlagPrefixTag, "prefix-tag", "", "prefix tag")
	flags.StringVar(&Cfg.c2k.FlagSuffixTag, "suffix-tag", "", "suffix tag")
	flags.BoolVar(&Cfg.c2k.FlagPassingOnly, "passing-only", true, "passing only")
	flags.BoolVar(&Cfg.c2k.FlagWithGatewayAPI, "with-gateway-api", false, "with gateway api")

	flags.BoolVar(&Cfg.k2c.FlagK8SDefault, "k8s-default-sync", true,
		"If true, all valid services in K8S are synced by default. If false, "+
			"the service must be annotated properly to sync. In either case "+
			"an annotation can override the default")
	flags.StringVar(&Cfg.k2c.FlagK8SServicePrefix, "k8s-service-prefix", "",
		"A prefix to prepend to all services written to Kubernetes from Consul. "+
			"If this is not set then services will have no prefix.")
	flags.StringVar(&Cfg.k2c.FlagConsulServicePrefix, "consul-service-prefix", "",
		"A prefix to prepend to all services written to Consul from Kubernetes. "+
			"If this is not set then services will have no prefix.")
	flags.StringVar(&Cfg.k2c.FlagK8SWriteNamespace, "k8s-write-namespace", metav1.NamespaceDefault,
		"The Kubernetes namespace to write to for services from Consul. "+
			"If this is not set then it will default to the default namespace.")
	flags.StringVar(&Cfg.k2c.FlagConsulK8STag, "consul-k8s-tag", "k8s",
		"Tag value for K8S services registered in Consul")
	flags.StringVar(&Cfg.k2c.FlagConsulNodeName, "consul-node-name", "k8s-sync",
		"The Consul node name to register for catalog sync. Defaults to k8s-sync. To be discoverable "+
			"via DNS, the name should only contain alpha-numerics and dashes.")
	flags.DurationVar(&Cfg.k2c.FlagConsulWritePeriod, "consul-write-interval", 30*time.Second,
		"The interval to perform syncing operations creating Consul services, formatted "+
			"as a time.Duration. All changes are merged and write calls are only made "+
			"on this interval. Defaults to 30 seconds (30s).")
	flags.BoolVar(&Cfg.k2c.FlagSyncClusterIPServices, "sync-clusterip-services", true,
		"If true, all valid ClusterIP services in K8S are synced by default. If false, "+
			"ClusterIP services are not synced to Consul.")
	flags.BoolVar(&Cfg.k2c.FlagSyncLBEndpoints, "sync-lb-services-endpoints", false,
		"If true, LoadBalancer service endpoints instead of ingress addresses will be synced to Consul. If false, "+
			"LoadBalancer endpoints are not synced to Consul.")
	flags.StringVar(&Cfg.k2c.FlagNodePortSyncType, "node-port-sync-type", "ExternalOnly",
		"Defines the type of sync for NodePort services. Valid options are ExternalOnly, "+
			"InternalOnly and ExternalFirst.")
	flags.BoolVar(&Cfg.k2c.FlagAddK8SNamespaceSuffix, "add-k8s-namespace-suffix", false,
		"If true, Kubernetes namespace will be appended to service names synced to Consul separated by a dash. "+
			"If false, no suffix will be appended to the service names in Consul. "+
			"If the service name annotation is provided, the suffix is not appended.")

	flags.Var((*AppendSliceValue)(&Cfg.k2c.FlagAllowK8sNamespacesList), "allow-k8s-namespace",
		"K8s namespaces to explicitly allow. May be specified multiple times.")
	flags.Var((*AppendSliceValue)(&Cfg.k2c.FlagDenyK8sNamespacesList), "deny-k8s-namespace",
		"K8s namespaces to explicitly deny. Takes precedence over allow. May be specified multiple times.")
	flags.BoolVar(&Cfg.k2c.FlagEnableNamespaces, "enable-namespaces", false,
		"[Enterprise Only] Enables namespaces, in either a single Consul namespace or mirrored.")
	flags.StringVar(&Cfg.k2c.FlagConsulDestinationNamespace, "consul-destination-namespace", "default",
		"[Enterprise Only] Defines which Consul namespace to register all synced services into. If '-enable-k8s-namespace-mirroring' "+
			"is true, this is not used.")
	flags.BoolVar(&Cfg.k2c.FlagEnableK8SNSMirroring, "enable-k8s-namespace-mirroring", false, "[Enterprise Only] Enables "+
		"namespace mirroring.")
	flags.StringVar(&Cfg.k2c.FlagK8SNSMirroringPrefix, "k8s-namespace-mirroring-prefix", "",
		"[Enterprise Only] Prefix that will be added to all k8s namespaces mirrored into Consul if mirroring is enabled.")
	flags.StringVar(&Cfg.k2c.FlagCrossNamespaceACLPolicy, "consul-cross-namespace-acl-policy", "",
		"[Enterprise Only] Name of the ACL policy to attach to all created Consul namespaces to allow service "+
			"discovery across Consul namespaces. Only necessary if ACLs are enabled.")

	flags.BoolVar(&Cfg.k2c.FlagEnableIngress, "enable-ingress", false,
		"[Enterprise Only] Enables namespaces, in either a single Consul namespace or mirrored.")
	flags.BoolVar(&Cfg.k2c.FlagLoadBalancerIPs, "loadBalancer-ips", false,
		"[Enterprise Only] Enables namespaces, in either a single Consul namespace or mirrored.")
}

// ValidateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func ValidateCLIParams() error {
	if Cfg.MeshName == "" {
		return fmt.Errorf("please specify the mesh name using --mesh-name")
	}

	if Cfg.FsmNamespace == "" {
		return fmt.Errorf("please specify the FSM namespace using -fsm-namespace")
	}

	if Cfg.DeriveNamespace == "" {
		return fmt.Errorf("please specify the cloud derive namespace using -derive-namespace")
	}

	if Cfg.SdrProvider == "" || (connector.EurekaDiscoveryService != Cfg.SdrProvider && connector.ConsulDiscoveryService != Cfg.SdrProvider) {
		return fmt.Errorf("please specify service discovery and registration provider using -sdr-provider")
	}

	if Cfg.HttpAddr == "" {
		return fmt.Errorf("please specify service discovery and registration server address using -sdr-http-addr")
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
