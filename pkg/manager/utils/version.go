package utils

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/flomesh-io/fsm/pkg/utils"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/repo"
)

// UpdateMainVersion updates main version of ingress controller
func UpdateMainVersion(basepath string, repoClient *repo.PipyRepoClient, _ configurator.Configurator) error {
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return err
	}

	oldVersion := gjson.Get(json, "version").String()
	newVersion := utils.SimpleHash(json)
	if oldVersion == newVersion {
		return nil
	}

	newJSON, err := sjson.Set(json, "version", newVersion)
	if err != nil {
		log.Error().Msgf("Failed to update HTTP config: %s", err)
		return err
	}

	return updateMainJSON(basepath, repoClient, newJSON)
}
