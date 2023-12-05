package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	Cfg   Config
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

type C2KCfg struct {
	FlagPassingOnly bool
	FlagFilterTag   string
	FlagPrefixTag   string
	FlagSuffixTag   string

	FlagWithGatewayEgress ViaFgwEgressCfg
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

type ViaFgwIngressCfg struct {
	Enable         bool
	ViaIngressType string
	ViaIngressPort uint
}

type ViaFgwEgressCfg struct {
	Enable        bool
	ViaEgressPort uint
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
	FlagAllowK8SNamespaces             []string // K8s namespaces to explicitly inject
	FlagDenyK8SNamespaces              []string // K8s namespaces to deny injection (has precedence)

	Consul K2ConsulCfg

	FlagWithGatewayIngress ViaFgwIngressCfg
}

type K2GCfg struct {
	FlagDefaultSync bool
	FlagSyncPeriod  time.Duration

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
	DeriveNamespace   string
	SdrProvider       string
	HttpAddr          string
	SyncCloudToK8s    bool
	SyncK8sToCloud    bool
	SyncK8sToGateway  bool

	C2K C2KCfg
	K2C K2CCfg
	K2G K2GCfg
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
	flags.StringVar(&Cfg.SdrProvider, "sdr-provider", "", "service discovery and registration (eureka, Consul)")
	flags.StringVar(&Cfg.HttpAddr, "sdr-http-addr", "", "http addr")

	flags.BoolVar(&Cfg.SyncCloudToK8s, "sync-cloud-to-k8s", true, "sync from cloud to k8s")
	flags.StringVar(&Cfg.C2K.FlagFilterTag, "sync-cloud-to-k8s-filter-tag", "", "filter tag")
	flags.StringVar(&Cfg.C2K.FlagPrefixTag, "sync-cloud-to-k8s-prefix-tag", "", "prefix tag")
	flags.StringVar(&Cfg.C2K.FlagSuffixTag, "sync-cloud-to-k8s-suffix-tag", "", "suffix tag")
	flags.BoolVar(&Cfg.C2K.FlagPassingOnly, "sync-cloud-to-k8s-passing-only", true, "passing only")
	flags.BoolVar(&Cfg.C2K.FlagWithGatewayEgress.Enable, "sync-cloud-to-k8s-with-gateway-egress", false, "with gateway api")
	flags.UintVar(&Cfg.C2K.FlagWithGatewayEgress.ViaEgressPort, "sync-cloud-to-k8s-with-gateway-egress-via-egress-port", 10090,
		"with gateway api via egress port")

	flags.BoolVar(&Cfg.SyncK8sToCloud, "sync-k8s-to-cloud", true, "sync from k8s to cloud")
	flags.BoolVar(&Cfg.K2C.FlagDefaultSync, "sync-k8s-to-cloud-default-sync", true,
		"If true, all valid services in K8S are synced by default. If false, "+
			"the service must be annotated properly to sync. In either case "+
			"an annotation can override the default")
	flags.DurationVar(&Cfg.K2C.FlagSyncPeriod, "sync-k8s-to-cloud-sync-interval", 5*time.Second,
		"The interval to perform syncing operations creating cloud services, formatted "+
			"as a time.Duration. All changes are merged and write calls are only made "+
			"on this interval. Defaults to 5 seconds (5s).")
	flags.BoolVar(&Cfg.K2C.FlagSyncClusterIPServices, "sync-k8s-to-cloud-sync-cluster-ip-services", true,
		"If true, all valid ClusterIP services in K8S are synced by default. If false, "+
			"ClusterIP services are not synced to Consul.")
	flags.BoolVar(&Cfg.K2C.FlagSyncLoadBalancerEndpoints, "sync-k8s-to-cloud-sync-load-balancer-services-endpoints", false,
		"If true, LoadBalancer service endpoints instead of ingress addresses will be synced to cloud. If false, "+
			"LoadBalancer endpoints are not synced to cloud.")
	flags.StringVar(&Cfg.K2C.FlagNodePortSyncType, "sync-k8s-to-cloud-node-port-sync-type", "ExternalOnly",
		"Defines the type of sync for NodePort services. Valid options are ExternalOnly, "+
			"InternalOnly and ExternalFirst.")

	flags.StringVar(&Cfg.K2C.FlagAddServicePrefix, "sync-k8s-to-cloud-add-service-prefix", "",
		"A prefix to prepend to all services written to cloud from Kubernetes. "+
			"If this is not set then services will have no prefix.")
	flags.BoolVar(&Cfg.K2C.FlagAddK8SNamespaceAsServiceSuffix, "sync-k8s-to-cloud-add-k8s-namespace-as-service-suffix", false,
		"If true, Kubernetes namespace will be appended to service names synced to cloud separated by a dash. "+
			"If false, no suffix will be appended to the service names in cloud. "+
			"If the service name annotation is provided, the suffix is not appended.")
	flags.Var((*AppendSliceValue)(&Cfg.K2C.FlagAllowK8SNamespaces), "sync-k8s-to-cloud-allow-k8s-namespaces",
		"K8s namespaces to explicitly allow. May be specified multiple times.")
	flags.Var((*AppendSliceValue)(&Cfg.K2C.FlagDenyK8SNamespaces), "sync-k8s-to-cloud-deny-k8s-namespaces",
		"K8s namespaces to explicitly deny. Takes precedence over allow. May be specified multiple times.")

	flags.BoolVar(&Cfg.K2C.FlagSyncIngress, "sync-k8s-to-cloud-sync-ingress", false,
		"enables syncing of the hostname from an Ingress resource to the service registration if an Ingress rule matches the service.")
	flags.BoolVar(&Cfg.K2C.FlagSyncIngressLoadBalancerIPs, "sync-k8s-to-cloud-sync-ingress-load-balancer-ips", false,
		"enables syncing the IP of the Ingress LoadBalancer if we do not want to sync the hostname from the Ingress resource.")

	flags.BoolVar(&Cfg.K2C.FlagWithGatewayIngress.Enable, "sync-k8s-to-cloud-with-gateway-ingress", false, "with gateway api")
	flags.StringVar(&Cfg.K2C.FlagWithGatewayIngress.ViaIngressType, "sync-k8s-to-cloud-with-gateway-ingress-via-ingress-type", "ClusterIP",
		"with gateway api via ingress ClusterIP/ExternalIP")
	flags.UintVar(&Cfg.K2C.FlagWithGatewayIngress.ViaIngressPort, "sync-k8s-to-cloud-with-gateway-ingress-via-ingress-port", 10080,
		"with gateway api via ingress port")

	flags.StringVar(&Cfg.K2C.Consul.FlagConsulNodeName, "sync-k8s-to-cloud-consul-node-name", "k8s-sync",
		"The Consul node name to register for catalog sync. Defaults to k8s-sync. To be discoverable "+
			"via DNS, the name should only contain alpha-numerics and dashes.")
	flags.StringVar(&Cfg.K2C.Consul.FlagConsulK8STag, "sync-k8s-to-cloud-consul-k8s-tag", "k8s",
		"Tag value for K8S services registered in cloud")
	flags.BoolVar(&Cfg.K2C.Consul.FlagConsulEnableNamespaces, "sync-k8s-to-cloud-consul-enable-namespaces", false,
		"[Enterprise Only] Enables namespaces, in either a single Consul namespace or mirrored.")
	flags.StringVar(&Cfg.K2C.Consul.FlagConsulDestinationNamespace, "sync-k8s-to-cloud-consul-destination-namespace", "default",
		"[Enterprise Only] Defines which Consul namespace to register all synced services into. If '-sync-k8s-to-cloud-consul-enable-k8s-namespace-mirroring' "+
			"is true, this is not used.")
	flags.BoolVar(&Cfg.K2C.Consul.FlagConsulEnableK8SNSMirroring, "sync-k8s-to-cloud-consul-enable-k8s-namespace-mirroring", false, "[Enterprise Only] Enables "+
		"namespace mirroring.")
	flags.StringVar(&Cfg.K2C.Consul.FlagConsulK8SNSMirroringPrefix, "sync-k8s-to-cloud-consul-k8s-namespace-mirroring-prefix", "",
		"[Enterprise Only] Prefix that will be added to all k8s namespaces mirrored into Consul if mirroring is enabled.")
	flags.StringVar(&Cfg.K2C.Consul.FlagConsulCrossNamespaceACLPolicy, "sync-k8s-to-cloud-consul-cross-namespace-acl-policy", "",
		"[Enterprise Only] Name of the ACL policy to attach to all created Consul namespaces to allow service "+
			"discovery across Consul namespaces. Only necessary if ACLs are enabled.")

	flags.BoolVar(&Cfg.SyncK8sToGateway, "sync-k8s-to-fgw", true, "sync from k8s to fgw")
	flags.BoolVar(&Cfg.K2G.FlagDefaultSync, "sync-k8s-to-fgw-default-sync", true,
		"If true, all valid services in K8S are synced by default. If false, "+
			"the service must be annotated properly to sync. In either case "+
			"an annotation can override the default")
	flags.DurationVar(&Cfg.K2G.FlagSyncPeriod, "sync-k8s-to-fgw-sync-interval", 5*time.Second,
		"The interval to perform syncing operations creating cloud services, formatted "+
			"as a time.Duration. All changes are merged and write calls are only made "+
			"on this interval. Defaults to 5 seconds (5s).")
	flags.Var((*AppendSliceValue)(&Cfg.K2G.FlagAllowK8SNamespaces), "sync-k8s-to-fgw-allow-k8s-namespaces",
		"K8s namespaces to explicitly allow. May be specified multiple times.")
	flags.Var((*AppendSliceValue)(&Cfg.K2G.FlagDenyK8SNamespaces), "sync-k8s-to-fgw-deny-k8s-namespaces",
		"K8s namespaces to explicitly deny. Takes precedence over allow. May be specified multiple times.")
}

// ValidateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func ValidateCLIParams() error {
	bytes, _ := json.MarshalIndent(Cfg, "", " ")
	fmt.Println(string(bytes))
	if len(Cfg.MeshName) == 0 {
		return fmt.Errorf("please specify the mesh name using --mesh-name")
	}

	if len(Cfg.FsmNamespace) == 0 {
		return fmt.Errorf("please specify the FSM namespace using -fsm-namespace")
	}

	if len(Cfg.SdrProvider) > 0 {
		if connector.EurekaDiscoveryService != Cfg.SdrProvider &&
			connector.ConsulDiscoveryService != Cfg.SdrProvider &&
			connector.MachineDiscoveryService != Cfg.SdrProvider {
			return fmt.Errorf("please specify service discovery and registration provider using -sdr-provider")
		}
	}

	if connector.EurekaDiscoveryService == Cfg.SdrProvider || connector.ConsulDiscoveryService == Cfg.SdrProvider {
		if len(Cfg.HttpAddr) == 0 {
			return fmt.Errorf("please specify service discovery and registration server address using -sdr-http-addr")
		}
		if Cfg.SyncCloudToK8s || Cfg.SyncK8sToCloud {
			if len(Cfg.DeriveNamespace) == 0 {
				return fmt.Errorf("please specify the cloud derive namespace using -derive-namespace")
			}
		}
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
