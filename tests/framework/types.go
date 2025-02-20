package framework

import (
	"bufio"
	"bytes"
	"io"
	"time"

	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	extClientset "github.com/flomesh-io/fsm/pkg/gen/client/extension/clientset/versioned"
	policyAttachmentClientset "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"

	"github.com/onsi/ginkgo"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kind/pkg/cluster"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	versioned2 "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/gen/client/policy/clientset/versioned"

	k3dCfg "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"

	"github.com/flomesh-io/fsm/pkg/cli"
)

// OSCrossPlatform indicates that a test can run on all operating systems.
const OSCrossPlatform string = "Cross-platform"

// FSMDescribeInfo is a struct to represent the Tier and Bucket of a given e2e test
type FSMDescribeInfo struct {
	// Tier represents the priority of the test. Lower value indicates higher priority.
	Tier int

	// Bucket indicates in which test Bucket the test will run in for CI. Each
	// Bucket is run in parallel while tests in the same Bucket run sequentially.
	Bucket int

	// OS indicates which OS the test can run on.
	OS string
}

// InstallType defines several FSM test deployment scenarios
type InstallType string

// CollectLogsType defines if/when to collect logs
type CollectLogsType string

// FsmTestData stores common state, variables and flags for the test at hand
type FsmTestData struct {
	T           ginkgo.GinkgoTInterface // for common test logging
	TestID      uint64                  // uint randomized for every test. GinkgoRandomSeed can't be used as is per-suite.
	TestDirBase string                  // Test directory base, default "/tmp", can be variable overridden
	TestDirName string                  // Autogenerated, based on test ID

	CleanupTest          bool            // Cleanup test-related resources once finished
	WaitForCleanup       bool            // Forces test to wait for effective deletion of resources upon cleanup
	IgnoreRestarts       bool            // Ignore control plane processes restarts, if any
	InstType             InstallType     // Install type
	CollectLogs          CollectLogsType // Collect logs type
	InitialRestartValues map[string]int  // Captures properly if an FSM instance have restarted during a NoInstall test

	FsmNamespace      string
	FsmMeshConfigName string
	FsmImageTag       string
	EnableNsMetricTag bool

	// Container registry related vars
	CtrRegistryUser     string // registry login
	CtrRegistryPassword string // registry password, if any
	CtrRegistryServer   string // server name. Has to be network reachable

	// Kind && cluster related vars
	ClusterName                string // Kind/K3d cluster name (used if kindCluster/k3dCluster)
	CleanupClusterBetweenTests bool   // Clean and re-create kind/k3d cluster between tests
	CleanupCluster             bool   // Cleanup kind/k3d cluster upon test finish
	ClusterVersion             string // Kind/K3d cluster version, ex. v1.20.2

	ClusterOS string // The operating system of the working nodes in the cluster. Mixed OS traffic is not supported.

	ReqSuccessTimeout time.Duration // ReqSuccessTimeout timeout duration that the test expects for all requests from the client to server to succeed.

	// Cluster handles and rest config
	Env             *cli.EnvSettings
	RestConfig      *rest.Config
	Client          *kubernetes.Clientset
	APIServerClient *clientset.Clientset

	SmiClients *smiClients

	// FSM's API clients
	PolicyClient           *versioned.Clientset
	ConfigClient           *versioned2.Clientset
	GatewayAPIClient       gatewayApiClientset.Interface
	NsigClient             nsigClientset.Interface
	ExtensionClient        extClientset.Interface
	PolicyAttachmentClient policyAttachmentClientset.Interface

	ClusterProvider *cluster.Provider // provider, used when kindCluster is used

	ClusterConfig *k3dCfg.ClusterConfig

	DeployOnOpenShift bool // Determines whether to configure tests for OpenShift

	RetryAppPodCreation bool // Whether to retry app pod creation due to issue #3973
}

// InstallFSMOpts describes install options for FSM
type InstallFSMOpts struct {
	ControlPlaneNS          string
	CertManager             string
	ContainerRegistryLoc    string
	ContainerRegistrySecret string
	FsmImageTag             string
	DeployGrafana           bool
	DeployPrometheus        bool
	DeployJaeger            bool
	DeployFluentbit         bool
	EnableReconciler        bool
	EnableIngress           bool
	IngressHTTPPort         int32
	EnableIngressTLS        bool
	IngressTLSPort          int32
	EnableNamespacedIngress bool
	EnableGateway           bool
	EnableFLB               bool
	EnableServiceLB         bool
	EnableEgressGateway     bool

	VaultHost            string
	VaultProtocol        string
	VaultPort            int
	VaultToken           string
	VaultRole            string
	VaultTokenSecretName string
	VaultTokenSecretKey  string

	CertmanagerIssuerGroup string
	CertmanagerIssuerKind  string
	CertmanagerIssuerName  string
	CertKeyBitSize         int
	CertValidtyDuration    time.Duration

	EgressEnabled        bool
	EnablePermissiveMode bool
	FSMLogLevel          string
	SidecarLogLevel      string
	SidecarClass         string
	LocalProxyMode       configv1alpha3.LocalProxyMode

	SetOverrides []string

	EnablePrivilegedInitContainer bool
	EnableIngressBackendPolicy    bool
	EnableAccessControlPolicy     bool
	EnableRetryPolicy             bool

	TrafficInterceptionMode string
}

// InstallFsmOpt is a function type for setting install options
type InstallFsmOpt func(*InstallFSMOpts)

// CleanupType identifies what triggered the cleanup
type CleanupType string

// DockerConfig and other configs are docker-specific container registry secret structures.
// Most of it is taken or referenced from kubectl source itself
type DockerConfig map[string]DockerConfigEntry

// DockerConfigJSON  is a struct for docker-specific config
type DockerConfigJSON struct {
	Auths       DockerConfig      `json:"auths"`
	HTTPHeaders map[string]string `json:"HttpHeaders,omitempty"`
}

// DockerConfigEntry is a struct for docker-specific container registry secret structures
type DockerConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// SuccessFunction is a simple definition for a success function.
// True as success, false otherwise
type SuccessFunction func() bool

// RetryOnErrorFunc is a function type passed to RetryFuncOnError() to execute
type RetryOnErrorFunc func() error

type LogConsumerWriter struct {
	consumer func(string)
}

func (l LogConsumerWriter) Write(p []byte) (n int, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	scanner.Buffer(make([]byte, 64*1024), bufio.MaxScanTokenSize)
	for scanner.Scan() {
		l.consumer(scanner.Text())
	}
	return len(p), nil
}

var _ io.Writer = &LogConsumerWriter{}
