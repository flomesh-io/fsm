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

const namespaceIgnoreDescription = `
This command will prevent a namespace or a set of namespaces from
participating in the mesh. Automatic sidecar injection on pods
belonging to the given namespace or set of namespaces will be prevented.
The command will not remove previously injected sidecars on pods belonging
to the given namespaces.
`

type namespaceIgnoreCmd struct {
	out        io.Writer
	namespaces []string
	clientSet  kubernetes.Interface
}

func newNamespaceIgnore(out io.Writer) *cobra.Command {
	ignoreCmd := &namespaceIgnoreCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "ignore NAMESPACE ...",
		Short: "ignore namespace from participating in the mesh",
		Long:  namespaceIgnoreDescription,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ignoreCmd.namespaces = args
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			ignoreCmd.clientSet = clientset
			return ignoreCmd.run()
		},
	}

	return cmd
}

func (cmd *namespaceIgnoreCmd) run() error {
	for _, ns := range cmd.namespaces {
		ns = strings.TrimSpace(ns)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if _, err := cmd.clientSet.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{}); err != nil {
			return fmt.Errorf("Failed to retrieve namespace [%s]: %w", ns, err)
		}

		// Patch the namespace with ignore label
		patch := fmt.Sprintf(`
{
	"metadata": {
		"labels": {
			"%s": "true"
		}
	}
}`, constants.IgnoreLabel)

		_, err := cmd.clientSet.CoreV1().Namespaces().Patch(ctx, ns, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
		if err != nil {
			return fmt.Errorf("Failed to configure namespace [%s] to be ignored: %w", ns, err)
		}

		fmt.Fprintf(cmd.out, "Successfully configured namespace [%s] to be ignored\n", ns)
	}

	return nil
}
