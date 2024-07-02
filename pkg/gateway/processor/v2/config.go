package v2

import (
	"fmt"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/tidwall/gjson"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// BuildConfigs builds the configs for all the gateways in the processor
func (c *GatewayProcessor) BuildConfigs() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	syncConfig := func(gateway *gwv1.Gateway, config fgw.Config) {
		gatewayPath := utils.GatewayCodebasePath(gateway.Namespace, gateway.Name)
		if exists := c.repoClient.CodebaseExists(gatewayPath); !exists {
			return
		}

		jsonVersion, err := c.getVersionOfConfigJSON(gatewayPath)
		if err != nil {
			return
		}

		log.Debug().Msgf("jsonVersion: %q, config version: %q", jsonVersion, config.GetVersion())

		if jsonVersion == config.GetVersion() {
			// config not changed, ignore updating
			log.Debug().Msgf("%s/config.json doesn't change, ignore updating...", gatewayPath)
			return
		}

		batches := []repo.Batch{
			{
				Basepath: gatewayPath,
				Items: []repo.BatchItem{
					{Path: "", Filename: "config.json", Content: config},
				},
			},
		}

		if err := c.repoClient.Batch(batches); err != nil {
			log.Error().Msgf("Sync config of Gateway %s/%s to repo failed: %s", gateway.Namespace, gateway.Name, err)
			return
		}
	}

	for _, gw := range gwutils.GetActiveGateways(c.client) {
		cfg := NewGatewayConfigGenerator(gw, c, c.client).Generate()

		go syncConfig(gw, cfg)
	}
}

func (c *GatewayProcessor) getVersionOfConfigJSON(basepath string) (string, error) {
	path := fmt.Sprintf("%s/config.json", basepath)

	json, err := c.repoClient.GetFile(path)
	if err != nil {
		log.Error().Msgf("Get %q from pipy repo error: %s", path, err)
		return "", err
	}

	version := gjson.Get(json, "version").String()

	return version, nil
}
