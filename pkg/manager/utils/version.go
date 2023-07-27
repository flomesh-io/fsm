package utils

import (
	"github.com/flomesh-io/fsm/pkg/configurator"
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/tidwall/sjson"
	"k8s.io/klog/v2"
	"time"
)

func UpdateMainVersion(basepath string, repoClient *repo.PipyRepoClient, mc configurator.Configurator) error {
	json, err := getMainJson(basepath, repoClient)
	if err != nil {
		return err
	}

	newJson, err := sjson.Set(json, "version", time.Now().UnixMilli())
	if err != nil {
		klog.Errorf("Failed to update HTTP config: %s", err)
		return err
	}

	return updateMainJson(basepath, repoClient, newJson)
}
