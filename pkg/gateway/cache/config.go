package cache

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	"github.com/tidwall/gjson"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// BuildConfigs builds the configs for all the gateways in the cache
func (c *GatewayCache) BuildConfigs() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	policies := c.policyAttachments()
	referenceGrants := c.getResourcesFromCache(informers.ReferenceGrantResourceType, false)

	for _, gw := range c.getActiveGateways() {
		processor := &GatewayProcessor{
			cache:           c,
			gateway:         gw,
			policies:        policies,
			referenceGrants: referenceGrants,
			validListeners:  gwutils.GetValidListenersFromGateway(gw),
			services:        make(map[string]serviceContext),
			rules:           make(map[int32]fgw.RouteRule),
		}

		cfg := processor.build()

		go func(gateway types.NamespacedName, config *fgw.ConfigSpec) {
			gatewayPath := utils.GatewayCodebasePath(gateway.Namespace)
			if exists := c.repoClient.CodebaseExists(gatewayPath); !exists {
				return
			}

			jsonVersion, err := c.getVersionOfConfigJSON(gatewayPath)
			if err != nil {
				return
			}

			if jsonVersion == cfg.Version {
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
		}(client.ObjectKeyFromObject(gw), cfg)
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
