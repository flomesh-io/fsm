package v2

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/dump"

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

	for _, gw := range gwutils.GetActiveGateways(c.client) {
		cfg := NewGatewayConfigGenerator(gw, c, c.client).Generate()

		if c.cfg.GetFeatureFlags().GenerateSingleConfigForFGW {
			go c.syncConfig(gw, cfg)
		} else {
			go c.syncConfigDir(gw, cfg)
		}
	}
}

func (c *GatewayProcessor) syncConfig(gateway *gwv1.Gateway, config fgw.Config) {
	gatewayPath := utils.GatewayCodebasePath(gateway.Namespace, gateway.Name)
	if exists := c.repoClient.CodebaseExists(gatewayPath); !exists {
		return
	}

	jsonVersion, err := c.getVersion(gatewayPath, "config.json")
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

func (c *GatewayProcessor) syncConfigDir(gateway *gwv1.Gateway, config fgw.Config) {
	gatewayPath := utils.GatewayCodebasePath(gateway.Namespace, gateway.Name)
	if exists := c.repoClient.CodebaseExists(gatewayPath); !exists {
		return
	}

	jsonVersion, err := c.getVersion(gatewayPath, "version.json")
	if err != nil {
		return
	}

	log.Debug().Msgf("jsonVersion: %q, config version: %q", jsonVersion, config.GetVersion())

	if jsonVersion == config.GetVersion() {
		// config not changed, ignore updating
		log.Debug().Msgf("%s/version.json doesn't change, ignore updating...", gatewayPath)
		return
	}

	batch := repo.Batch{
		Basepath: gatewayPath,
		Items: []repo.BatchItem{
			{Path: "", Filename: "version.json", Content: fmt.Sprintf(`{"version": "%s"}`, config.GetVersion())},
		},
	}

	resourceName := func(r fgw.Resource) string {
		if len(r.GetNamespace()) == 0 {
			return fmt.Sprintf("%s-%s.yaml", r.GetKind(), r.GetName())
		}

		return fmt.Sprintf("%s-%s-%s.yaml", r.GetKind(), r.GetNamespace(), r.GetName())
	}

	for _, r := range config.GetResources() {
		batch.Items = append(batch.Items,
			repo.BatchItem{
				Path:     "/resources",
				Filename: resourceName(r),
				Content:  toYAML(r),
			},
		)
	}

	for k, v := range config.GetSecrets() {
		batch.Items = append(batch.Items,
			repo.BatchItem{
				Path:     "/secrets",
				Filename: k,
				Content:  v,
			},
		)
	}

	for t, filters := range config.GetFilters() {
		for k, v := range filters {
			batch.Items = append(batch.Items,
				repo.BatchItem{
					Path:     fmt.Sprintf("/filters/%s", strings.ToLower(string(t))),
					Filename: fmt.Sprintf("%s.js", k),
					Content:  v,
				},
			)
		}
	}

	if err := c.repoClient.Batch([]repo.Batch{batch}); err != nil {
		log.Error().Msgf("Sync config of Gateway %s/%s to repo failed: %s", gateway.Namespace, gateway.Name, err)
		return
	}
}

func (c *GatewayProcessor) getVersion(basepath string, file string) (string, error) {
	path := fmt.Sprintf("%s/%s", basepath, file)

	json, err := c.repoClient.GetFile(path)
	if err != nil {
		log.Error().Msgf("Get %q from pipy repo error: %s", path, err)
		return "", err
	}

	version := gjson.Get(json, "version").String()

	return version, nil
}

func toYAML(v interface{}) string {
	y, err := yaml.Marshal(v)
	if err != nil {
		log.Error().Msgf("yaml marshal failed:%v\n%v\n", err, dump.Pretty(v))
		return ""
	}

	return string(y)
}
