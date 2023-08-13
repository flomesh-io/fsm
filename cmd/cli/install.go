package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	_ "embed" // required to embed resources
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/yaml"

	"github.com/flomesh-io/fsm/pkg/cli"
)

const installDesc = `
This command installs an fsm control plane on the Kubernetes cluster.

An fsm control plane is comprised of namespaced Kubernetes resources
that get installed into the fsm-system namespace as well as cluster
wide Kubernetes resources.

The default Kubernetes namespace that gets created on install is called
fsm-system. To create an install control plane components in a different
namespace, use the global --fsm-namespace flag.

Example:
  $ fsm install --fsm-namespace hello-world

Multiple control plane installations can exist within a cluster. Each
control plane is given a cluster-wide unqiue identifier called mesh name.
A mesh name can be passed in via the --mesh-name flag. By default, the
mesh-name name will be set to "fsm." The mesh name must conform to same
guidelines as a valid Kubernetes label value. Must be 63 characters or
less and must be empty or begin and end with an alphanumeric character
([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and
alphanumerics between.

Example:
  $ fsm install --mesh-name "hello-fsm"

The mesh name is used in various ways like for naming Kubernetes resources as
well as for adding a Kubernetes Namespace to the list of Namespaces a control
plane should watch for sidecar injection of proxies.
`
const (
	defaultChartPath         = ""
	defaultMeshName          = "fsm"
	defaultEnforceSingleMesh = true
)

// chartTGZSource is the `helm package`d representation of the default Helm chart.
// Its value is embedded at build time.
//
//go:embed chart.tgz
var chartTGZSource []byte

type installCmd struct {
	out            io.Writer
	chartPath      string
	meshName       string
	timeout        time.Duration
	clientSet      kubernetes.Interface
	chartRequested *chart.Chart
	setOptions     []string // --set
	atomic         bool
	// Toggle this to enforce only one mesh in this cluster
	enforceSingleMesh bool
	disableSpinner    bool

	valueFiles []string // -f/--values
}

func newInstallCmd(config *helm.Configuration, out io.Writer) *cobra.Command {
	inst := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install fsm control plane",
		Long:  installDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			clientset, err := kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			inst.clientSet = clientset
			return inst.run(config)
		},
	}

	f := cmd.Flags()
	f.StringVar(&inst.chartPath, "fsm-chart-path", defaultChartPath, "path to fsm chart to override default chart")
	f.StringVar(&inst.meshName, "mesh-name", defaultMeshName, "name for the new control plane instance")
	f.BoolVar(&inst.enforceSingleMesh, "enforce-single-mesh", defaultEnforceSingleMesh, "Enforce only deploying one mesh in the cluster")
	f.DurationVar(&inst.timeout, "timeout", 5*time.Minute, "Time to wait for installation and resources in a ready state, zero means no timeout")
	f.StringArrayVar(&inst.setOptions, "set", nil, "Set arbitrary chart values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	f.BoolVar(&inst.atomic, "atomic", false, "Automatically clean up resources if installation fails")
	f.StringSliceVarP(&inst.valueFiles, "values", "f", []string{}, "Specify values in a YAML file (can specify multiple)")

	return cmd
}

func (i *installCmd) run(config *helm.Configuration) error {
	if err := i.loadFSMChart(); err != nil {
		return err
	}

	values := map[string]interface{}{}
	// values represents the overrides for the FSM chart's values.yaml file
	setValues, err := i.resolveValues()
	if err != nil {
		return err
	}
	debug("setValues: %s", setValues)

	fileValues, err := i.resoleValueFiles()
	if err != nil {
		return err
	}
	debug("fileValues: %s", fileValues)

	// --set takes precedence over --values/-f
	values = mergeMaps(values, fileValues)
	values = mergeMaps(values, setValues)

	debug("values: %s", values)

	installClient := helm.NewInstall(config)
	installClient.ReleaseName = i.meshName
	installClient.Namespace = settings.Namespace()
	installClient.CreateNamespace = true
	installClient.Wait = true
	installClient.Atomic = i.atomic
	installClient.Timeout = i.timeout

	debug("Beginning FSM installation")
	if i.disableSpinner || settings.Verbose() {
		if _, err = installClient.Run(i.chartRequested, values); err != nil {
			if !settings.Verbose() {
				return err
			}

			pods, _ := i.clientSet.CoreV1().Pods(settings.Namespace()).List(context.Background(), metav1.ListOptions{})

			for _, pod := range pods.Items {
				fmt.Fprintf(i.out, "Status for pod %s in namespace %s:\n %v\n\n", pod.Name, pod.Namespace, pod.Status)
			}
			return err
		}
	} else {
		spinner := new(cli.Spinner)
		spinner.Init(i.clientSet, settings.Namespace(), values)
		err = spinner.Run(func() error {
			_, installErr := installClient.Run(i.chartRequested, values)
			return installErr
		})
		if err != nil {
			if !settings.Verbose() {
				return err
			}
		}
	}
	fmt.Fprintf(i.out, "FSM installed successfully in namespace [%s] with mesh name [%s]\n", settings.Namespace(), i.meshName)
	return nil
}

func (i *installCmd) loadFSMChart() error {
	debug("Loading FSM helm chart")
	var err error
	if i.chartPath != "" {
		i.chartRequested, err = loader.Load(i.chartPath)
	} else {
		i.chartRequested, err = loader.LoadArchive(bytes.NewReader(chartTGZSource))
	}

	if err != nil {
		return fmt.Errorf("error loading chart for installation: %w", err)
	}

	return nil
}

func (i *installCmd) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	if err := parseVal(i.setOptions, finalValues); err != nil {
		return nil, fmt.Errorf("invalid format for --set: %w", err)
	}

	valuesConfig := []string{
		fmt.Sprintf("fsm.meshName=%s", i.meshName),
		fmt.Sprintf("fsm.enforceSingleMesh=%t", i.enforceSingleMesh),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return nil, err
	}

	return finalValues, nil
}

// parses Helm strvals line and merges into a map
func parseVal(vals []string, parsedVals map[string]interface{}) error {
	for _, v := range vals {
		if err := strvals.ParseInto(v, parsedVals); err != nil {
			return err
		}
	}
	return nil
}

func (i *installCmd) resoleValueFiles() (map[string]interface{}, error) {
	base := map[string]interface{}{}

	// User specified a values files via -f/--values
	for _, filePath := range i.valueFiles {
		currentMap := map[string]interface{}{}

		valueBytes, err := readFile(filePath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(valueBytes, &currentMap); err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s", filePath)
		}
		// Merge with the previous map
		base = mergeMaps(base, currentMap)
	}

	return base, nil
}

// readFile load a file from stdin, the local directory, or a remote file with a url.
func readFile(filePath string) ([]byte, error) {
	if strings.TrimSpace(filePath) == "-" {
		return io.ReadAll(os.Stdin)
	}

	return os.ReadFile(filepath.Clean(filePath))
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
