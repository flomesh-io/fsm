package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/constants"
)

const metricsDisableDescription = `
This command will disable metrics scraping on all pods belonging to the given
namespace or set of namespaces.
`

type metricsDisableCmd struct {
	out        io.Writer
	namespaces []string
	clientSet  kubernetes.Interface
}

func newMetricsDisable(out io.Writer) *cobra.Command {
	disableCmd := &metricsDisableCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "disable ...",
		Short: "disable metrics",
		Long:  metricsDisableDescription,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			disableCmd.clientSet = clientset
			return disableCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringSliceVar(&disableCmd.namespaces, "namespace", []string{}, "One or more namespaces to disable metrics on")

	return cmd
}

func (cmd *metricsDisableCmd) run() error {
	// Add metrics annotation on namespaces
	for _, ns := range cmd.namespaces {
		ns = strings.TrimSpace(ns)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		namespace, err := cmd.clientSet.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Failed to retrieve namespace [%s]: %w", ns, err)
		}

		// Check if the namespace belongs to a mesh, if not return an error
		monitored, err := isMonitoredNamespace(*namespace, getMeshNames(cmd.clientSet))
		if err != nil {
			return err
		}
		if !monitored {
			return fmt.Errorf("Namespace [%s] does not belong to a mesh, missing annotation %q",
				ns, constants.FSMKubeResourceMonitorAnnotation)
		}

		// Patch the namespace to remove the metrics annotation.
		patch := fmt.Sprintf(`
{
	"metadata": {
		"annotations": {
			"%s": null
		}
	}
}`, constants.MetricsAnnotation)

		_, err = cmd.clientSet.CoreV1().Namespaces().Patch(ctx, ns, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
		if err != nil {
			return fmt.Errorf("Failed to disable metrics in namespace [%s]: %w", ns, err)
		}

		// Disable metrics on pods belonging to this namespace
		if err := cmd.disableMetricsForPods(ns); err != nil {
			return fmt.Errorf("Failed to disable metrics for existing pod in namespace [%s]: %w", ns, err)
		}

		fmt.Fprintf(cmd.out, "Metrics successfully disabled in namespace [%s]\n", ns)
	}

	return nil
}

// disableMetricsForPods disables metrics for existing pods in the given namespace
func (cmd *metricsDisableCmd) disableMetricsForPods(namespace string) error {
	listOptions := metav1.ListOptions{
		// Matches on pods which are already a part of the mesh, which contain the Sidecar ID label
		LabelSelector: constants.SidecarUniqueIDLabelName,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	podList, err := cmd.clientSet.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		// Patch existing pods in this namespace to remove metrics annotation
		patch := fmt.Sprintf(`
{
	"metadata": {
		"annotations": {
			"%s": null,
			"%s": null,
			"%s": null
		}
	}
}`, constants.PrometheusScrapeAnnotation, constants.PrometheusPortAnnotation, constants.PrometheusPathAnnotation)

		_, err = cmd.clientSet.CoreV1().Pods(namespace).Patch(ctx, pod.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
		if err != nil {
			return err
		}
	}

	return nil
}
