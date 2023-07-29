//lint:file-ignore U1000 Ignore all unused code, it's test code.

// Package scale implements scale test's methods.
package scale

import (
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
)

const (
	defaultFilename = "results.txt"
)

// Convenience function that wraps usual installation requirements for
// initializing a scale test (FSM install, prometheus/grafana deployment /w rendering, scale handle, etc.)
func scaleFSMInstall() (*DataHandle, error) {
	// Prometheus scrapping is not scalable past a certain number of proxies given
	// current configuration/constraints. We will disable getting proxy metrics
	// while we focus on qualifying control plane.
	// Note: this does not prevent fsm metrics scraping.
	Td.EnableNsMetricTag = false

	// Only Collect logs for control plane processess
	Td.CollectLogs = ControlPlaneOnly

	t := Td.GetFSMInstallOpts()
	// To avoid logging become a burden, use error logging as a regular setup would
	t.FSMLogLevel = "error"
	t.SidecarLogLevel = "error"

	// Override Memory available, both requested and limit to 1G to guarantee the memory available
	// for FSM will not depend on the Node's load.
	t.SetOverrides = append(t.SetOverrides,
		"fsm.fsmController.resource.requests.memory=1G")
	t.SetOverrides = append(t.SetOverrides,
		"fsm.fsmController.resource.limits.memory=1G")

	// enable Prometheus and Grafana, plus remote rendering
	t.DeployGrafana = true
	t.DeployPrometheus = true
	t.SetOverrides = append(t.SetOverrides,
		"fsm.grafana.enableRemoteRendering=true")

	err := Td.InstallFSM(t)
	if err != nil {
		return nil, err
	}

	// Required to happen here, as Prometheus and Grafana are deployed with FSM install
	pHandle, err := Td.GetFSMPrometheusHandle()
	if err != nil {
		return nil, err
	}
	gHandle, err := Td.GetFSMGrafanaHandle()
	if err != nil {
		return nil, err
	}

	// New test data handle. We set usual resources to track and Grafana dashboards to save.
	sd := NewDataHandle(pHandle, gHandle, getFSMTrackResources(), getFSMGrafanaSaveDashboards())
	// Set the file descriptors we want results to get written to
	sd.ResultsOut = getFSMTestOutputFiles()

	return sd, nil
}

// Returns the FSM grafana dashboards of interest to save after the test
func getFSMGrafanaSaveDashboards() []GrafanaPanel {
	return []GrafanaPanel{
		{
			Filename:  "cpu",
			Dashboard: MeshDetails,
			Panel:     CPUPanel,
		},
		{
			Filename:  "mem",
			Dashboard: MeshDetails,
			Panel:     MemRSSPanel,
		},
	}
}

// Returns labels to select FSM controller and FSM-installed Prometheus.
func getFSMTrackResources() []TrackedLabel {
	return []TrackedLabel{
		{
			Namespace: Td.FsmNamespace,
			Label: metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.AppLabel: constants.FSMControllerName,
				},
			},
		},
		{
			Namespace: Td.FsmNamespace,
			Label: metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.AppLabel: FsmPrometheusAppLabel,
				},
			},
		},
	}
}

// Get common outputs we are interested to print in (resultsFile and stdout basically)
func getFSMTestOutputFiles() []*os.File {
	fName := filepath.Clean(Td.GetTestFilePath(defaultFilename))
	f, err := os.Create(fName)
	if err != nil {
		fmt.Printf("Failed to open file: %v", err)
		return nil
	}

	return []*os.File{
		f,
		os.Stdout,
	}
}
