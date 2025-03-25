package main

import (
	"fmt"
	"io"

	mapset "github.com/deckarep/golang-set"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
)

const metricsDescription = `
This command consists of multiple subcommands related to managing metrics
associated with fsm.
`

func newMetricsCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "manage metrics",
		Long:  metricsDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newMetricsEnable(out))
	cmd.AddCommand(newMetricsDisable(out))

	return cmd
}

// isMonitoredNamespace returns true if the Namespace is correctly annotated for monitoring given a set of existing meshes
func isMonitoredNamespace(ns corev1.Namespace, meshList mapset.Set) (bool, error) {
	// Check if the namespace has the resource monitor annotation
	meshName, monitored := ns.Labels[constants.FSMKubeResourceMonitorAnnotation]
	if !monitored {
		return false, nil
	}
	if meshName == "" {
		return false, fmt.Errorf("label %q on namespace %q cannot be empty",
			constants.FSMKubeResourceMonitorAnnotation, ns.Name)
	}
	if !meshList.Contains(meshName) {
		return false, fmt.Errorf("invalid mesh name %q used with label %q on namespace %q, must be one of %v",
			meshName, constants.FSMKubeResourceMonitorAnnotation, ns.Name, meshList.ToSlice())
	}

	return true, nil
}
