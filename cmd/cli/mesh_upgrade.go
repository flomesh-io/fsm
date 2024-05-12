package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	helmutil "github.com/flomesh-io/fsm/pkg/helm"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
)

const upgradeDesc = `
This command upgrades an FSM control plane by upgrading the
underlying Helm release.

The mesh to upgrade is identified by its mesh name and namespace. If either were
overridden from the default for the "fsm install" command, the --mesh-name and
--fsm-namespace flags need to be specified.

Values from the current Helm release will NOT be carried over to the new
release. Use --set to pass any overridden values from the old release to the new
release.

Note: edits to resources NOT made by Helm or the FSM CLI may not persist after
"fsm mesh upgrade" is run.

Note: edits made to chart values that impact the preset-mesh-config will not
apply to the fsm-mesh-config, when "fsm mesh upgrade" is run. This means configuration
changes made to the fsm-mesh-config resource will persist through an upgrade
and any configuration changes needed can be done by patching this resource prior or
post an upgrade.

If any CustomResourceDefinitions (CRDs) are different between the installed
chart and the upgraded chart, the CRDs will be updated to include the latest versions.
Any corresponding custom resources that wish to reference the newer CRD version can
be updated post upgrade.
`

const meshUpgradeExample = `
# Upgrade the mesh with the default name in the fsm-system namespace, setting
# the image registry and tag to the defaults, and leaving all other values unchanged.
fsm mesh upgrade --fsm-namespace fsm-system
`

type meshUpgradeCmd struct {
	out io.Writer

	meshName string
	chart    *chart.Chart

	setOptions []string // --set
	valueFiles []string // -f/--values
}

func newMeshUpgradeCmd(config *helm.Configuration, out io.Writer) *cobra.Command {
	upg := &meshUpgradeCmd{
		out: out,
	}
	var chartPath string

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "upgrade fsm control plane",
		Long:    upgradeDesc,
		Example: meshUpgradeExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if chartPath != "" {
				var err error
				upg.chart, err = loader.Load(chartPath)
				if err != nil {
					return err
				}
			}

			return upg.run(config)
		},
	}

	f := cmd.Flags()

	f.StringVar(&upg.meshName, "mesh-name", defaultMeshName, "Name of the mesh to upgrade")
	f.StringVar(&chartPath, "fsm-chart-path", "", "path to fsm chart to override default chart")
	f.StringArrayVar(&upg.setOptions, "set", nil, "Set arbitrary chart values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	f.StringSliceVarP(&upg.valueFiles, "values", "f", []string{}, "Specify values in a YAML file (can specify multiple)")

	return cmd
}

func (u *meshUpgradeCmd) run(config *helm.Configuration) error {
	if u.chart == nil {
		var err error
		u.chart, err = loader.LoadArchive(bytes.NewReader(chartTGZSource))
		if err != nil {
			return err
		}
	}

	// values represents the overrides for the FSM chart's values.yaml file
	setValues, err := u.resolveValues()
	if err != nil {
		return err
	}
	debug("setValues: %s", setValues)

	fileValues, err := u.resoleValuesFromFiles()
	if err != nil {
		return err
	}
	debug("fileValues: %s", fileValues)

	// --set takes precedence over --values/-f
	values := helmutil.MergeMaps(fileValues, setValues)
	debug("values: %s", values)

	// Add the overlay values to be updated to the current release's values map
	//values, err := u.resolveValues()
	//if err != nil {
	//	return err
	//}

	upgradeClient := helm.NewUpgrade(config)
	upgradeClient.Wait = true
	upgradeClient.Timeout = 5 * time.Minute
	upgradeClient.ResetValues = true
	if _, err = upgradeClient.Run(u.meshName, u.chart, values); err != nil {
		return err
	}

	fmt.Fprintf(u.out, "FSM successfully upgraded mesh [%s] in namespace [%s]\n", u.meshName, settings.Namespace())
	return nil
}

func (u *meshUpgradeCmd) resolveValues() (map[string]interface{}, error) {
	vals := make(map[string]interface{})
	for _, val := range u.setOptions {
		if err := strvals.ParseInto(val, vals); err != nil {
			return nil, fmt.Errorf("invalid format for --set: %w", err)
		}
	}
	return vals, nil
}

func (u *meshUpgradeCmd) resoleValuesFromFiles() (map[string]interface{}, error) {
	base := map[string]interface{}{}

	// User specified a values files via -f/--values
	for _, filePath := range u.valueFiles {
		currentMap := map[string]interface{}{}

		valueBytes, err := helmutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(valueBytes, &currentMap); err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s", filePath)
		}
		// Merge with the previous map
		base = helmutil.MergeMaps(base, currentMap)
	}

	return base, nil
}
