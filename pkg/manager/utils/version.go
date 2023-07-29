package utils

import (
	"time"

	"github.com/tidwall/sjson"

	"github.com/flomesh-io/fsm/pkg/configurator"
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
)

// UpdateMainVersion updates main version of ingress controller
func UpdateMainVersion(basepath string, repoClient *repo.PipyRepoClient, _ configurator.Configurator) error {
	json, err := getMainJSON(basepath, repoClient)
	if err != nil {
		return err
	}

	newJSON, err := sjson.Set(json, "version", time.Now().UnixMilli())
	if err != nil {
		log.Error().Msgf("Failed to update HTTP config: %s", err)
		return err
	}

	return updateMainJSON(basepath, repoClient, newJSON)
}
