package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func meshUpgradeConfig() *action.Configuration {
	mem := driver.NewMemory()
	mem.SetNamespace(settings.Namespace())
	store := storage.Init(mem)

	return &action.Configuration{
		Releases: store,
		KubeClient: &kubefake.PrintingKubeClient{
			Out: ioutil.Discard,
		},
		Capabilities: helmCapabilities(),
		Log:          func(_ string, _ ...interface{}) {},
	}
}

func defaultMeshUpgradeCmd() *meshUpgradeCmd {
	chart, err := loader.Load(testChartPath)
	if err != nil {
		panic(err)
	}

	return &meshUpgradeCmd{
		out:      ioutil.Discard,
		meshName: defaultMeshName,
		chart:    chart,
	}
}

func TestMeshUpgradeDefault(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath
	i.disableSpinner = true
	err := i.run(config)
	a.Nil(err)

	u := defaultMeshUpgradeCmd()

	getVals := action.NewGetValues(config)
	getVals.AllValues = true
	upgraded, err := getVals.Run(u.meshName)
	a.Nil(err)

	meshName, err := chartutil.Values(upgraded).PathValue("fsm.meshName")
	a.Nil(err)
	a.Equal(defaultMeshName, meshName)
	err = u.run(config)
	a.Nil(err)
}

func TestMeshUpgradeOverridesInstallDefaults(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath
	i.disableSpinner = true

	err := i.run(config)
	a.Nil(err)

	u := defaultMeshUpgradeCmd()
	defaultImageRegVal, err := chartutil.Values(u.chart.Values).PathValue("fsm.image.registry")
	a.NoError(err)
	defaultImageReg := defaultImageRegVal.(string)
	upgradedImageReg := "upgraded-" + defaultImageReg
	u.setOptions = []string{"fsm.image.registry=" + upgradedImageReg}

	err = u.run(config)
	a.Nil(err)

	getVals := action.NewGetValues(config)
	getVals.AllValues = true
	upgraded, err := getVals.Run(u.meshName)
	a.Nil(err)

	fsmImageReg, err := chartutil.Values(upgraded).PathValue("fsm.image.registry")
	a.Nil(err)
	a.Equal(upgradedImageReg, fsmImageReg)

	// Successive upgrades overriddes image-tag values from the previous upgrade
	u = defaultMeshUpgradeCmd()
	err = u.run(config)
	a.Nil(err)

	upgraded, err = getVals.Run(u.meshName)
	a.Nil(err)

	fsmImageReg, err = chartutil.Values(upgraded).PathValue("fsm.image.registry")
	a.Nil(err)
	a.Equal(defaultImageReg, fsmImageReg)
}

func TestMeshUpgradeDropsInstallOverrides(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath
	i.disableSpinner = true
	i.setOptions = []string{
		"fsm.enableEgress=true",
		"fsm.image.registry=installed",
		"fsm.sidecar.sidecarLogLevel=trace",
	}

	err := i.run(config)
	a.Nil(err)

	u := defaultMeshUpgradeCmd()

	err = u.run(config)
	a.Nil(err)

	getVals := action.NewGetValues(config)
	getVals.AllValues = true
	upgraded, err := getVals.Run(u.meshName)
	a.Nil(err)

	// Values overridden at install should be the same as their defaults in the
	// chart after an upgrade that sets no values
	for _, valKey := range []string{"fsm.enableEgress", "fsm.image.registry", "fsm.sidecar.sidecarLogLevel"} {
		def, defErr := chartutil.Values(u.chart.Values).PathValue(valKey)
		upgradedVal, upgErr := chartutil.Values(upgraded).PathValue(valKey)
		a.Equal(def, upgradedVal)
		a.Equal(defErr, upgErr)
	}
}

func TestMeshUpgradeToModifiedChart(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath
	i.disableSpinner = true

	err := i.run(config)
	a.Nil(err)

	u := defaultMeshUpgradeCmd()

	// Upgrade to a "new version" of the chart with a new required value.
	// The upgrade itself will fail if the required value is not defined.
	_, exists := u.chart.Values["newRequired"]
	a.False(exists)
	u.chart.Values["newRequired"] = "anything"
	u.chart.Schema = []byte(`{"required": ["newRequired"]}`)

	// A value NOT set explicitly by `fsm install` that exists in the old chart
	oldNamespace, err := chartutil.Values(u.chart.Values).PathValue("fsm.namespace")
	a.Nil(err)
	newNamespace := fmt.Sprintf("new-%s", oldNamespace)
	u.chart.Values["fsm"].(map[string]interface{})["namespace"] = newNamespace

	err = u.run(config)
	a.Nil(err)

	getVals := action.NewGetValues(config)
	getVals.AllValues = true
	upgraded, err := getVals.Run(u.meshName)
	a.Nil(err)

	// When a default is changed in values.yaml, use the new one
	namespace, err := chartutil.Values(upgraded).PathValue("fsm.namespace")
	a.Nil(err)
	a.Equal(newNamespace, namespace)
}

func TestMeshUpgradeRemovedValue(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath
	i.disableSpinner = true
	err := i.run(config)
	a.NoError(err)

	u := defaultMeshUpgradeCmd()

	// Upgrade to a "new version" of the chart with a deleted value.
	_, err = chartutil.Values(u.chart.Values).PathValue("fsm.namespace")
	a.NoError(err)
	delete(u.chart.Values["fsm"].(map[string]interface{}), "namespace")
	// Schema only accepting the remaining values
	u.chart.Schema = []byte(`{"properties": {"fsm": {"properties": {"image": {}, "imagePullSecrets": {}}, "additionalProperties": false}}}`)

	err = u.run(config)
	a.NoError(err)
}
