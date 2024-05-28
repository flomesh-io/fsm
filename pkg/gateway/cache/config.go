package cache

import (
	"fmt"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/tidwall/gjson"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// BuildConfigs builds the configs for all the gateways in the cache
func (c *GatewayCache) BuildConfigs() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	syncConfig := func(gateway *gwv1.Gateway, config *fgw.ConfigSpec) {
		gatewayPath := utils.GatewayCodebasePath(gateway.Namespace)
		if exists := c.repoClient.CodebaseExists(gatewayPath); !exists {
			return
		}

		jsonVersion, err := c.getVersionOfConfigJSON(gatewayPath)
		if err != nil {
			return
		}

		if jsonVersion == config.Version {
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

	for _, gw := range c.getActiveGateways() {
		cfg := NewGatewayProcessor(c, gw).build()

		go syncConfig(gw, cfg)
	}
}

func (c *GatewayCache) getVersionOfConfigJSON(basepath string) (string, error) {
	path := fmt.Sprintf("%s/config.json", basepath)

	json, err := c.repoClient.GetFile(path)
	if err != nil {
		log.Error().Msgf("Get %q from pipy repo error: %s", path, err)
		return "", err
	}

	version := gjson.Get(json, "Version").String()

	return version, nil
}
