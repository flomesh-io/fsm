package main

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chartutil"
)

const (
	presetMeshConfigName    = "preset-mesh-config"
	presetMeshConfigJSONKey = "preset-mesh-config.json"
)

var (
	kubeVersion119 = &chartutil.KubeVersion{
		Version: fmt.Sprintf("v%s.%s.0", "1", "19"),
		Major:   "1",
		Minor:   "19",
	}

	//kubeVersion121 = &chartutil.KubeVersion{
	//	Version: fmt.Sprintf("v%s.%s.0", "1", "21"),
	//	Major:   "1",
	//	Minor:   "21",
	//}
)
