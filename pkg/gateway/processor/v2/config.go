package v2

import (
	"fmt"
	"strings"

	"github.com/flomesh-io/fsm/pkg/constants"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/dump"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/tidwall/gjson"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	mrepo "github.com/flomesh-io/fsm/pkg/manager/repo"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// BuildConfigs builds the configs for all the gateways in the processor
func (c *GatewayProcessor) BuildConfigs() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.preCheck() {
		return
	}

	for _, gw := range gwutils.GetActiveGateways(c.client) {
		cfg := NewGatewayConfigGenerator(gw, c, c.client, c.cfg).Generate()

		go c.syncConfigDir(gw, cfg)
	}
}

func (c *GatewayProcessor) preCheck() bool {
	if !c.repoClient.IsRepoUp() {
		log.Trace().Msg("Repo is not up, ignore ...")
		return false
	}

	if !c.repoClient.CodebaseExists(constants.DefaultGatewayBasePath) {
		if err := c.repoClient.Batch([]repo.Batch{mrepo.GatewaysBatch()}); err != nil {
			log.Error().Msgf("Failed to write gateway scripts to repo: %s", err)
			return false
		}
	}

	defaultGatewaysPath := utils.GetDefaultGatewaysPath()
	if !c.repoClient.CodebaseExists(defaultGatewaysPath) {
		if err := c.repoClient.DeriveCodebase(defaultGatewaysPath, constants.DefaultGatewayBasePath); err != nil {
			log.Error().Msgf("%q failed to derive codebase %q: %s", defaultGatewaysPath, constants.DefaultGatewayBasePath, err)
			return false
		}
	}

	return true
}

//func (c *GatewayProcessor) syncConfig(gateway *gwv1.Gateway, config fgw.Config) {
//	gatewayPath := utils.GatewayCodebasePath(gateway.Namespace, gateway.Name)
//	if exists := c.repoClient.CodebaseExists(gatewayPath); !exists {
//		return
//	}
//
//	jsonVersion, err := c.getVersion(gatewayPath, "config.json")
//	if err != nil {
//		return
//	}
//
//	log.Debug().Msgf("jsonVersion: %q, config version: %q", jsonVersion, config.GetVersion())
//
//	if jsonVersion == config.GetVersion() {
//		// config not changed, ignore updating
//		log.Debug().Msgf("%s/config.json doesn't change, ignore updating...", gatewayPath)
//		return
//	}
//
//	batches := []repo.Batch{
//		{
//			Basepath: gatewayPath,
//			Items: []repo.BatchItem{
//				{Path: "", Filename: "config.json", Content: config},
//			},
//		},
//	}
//
//	if err := c.repoClient.Batch(batches); err != nil {
//		log.Error().Msgf("Sync config of Gateway %s/%s to repo failed: %s", gateway.Namespace, gateway.Name, err)
//		return
//	}
//}

func (c *GatewayProcessor) syncConfigDir(gateway *gwv1.Gateway, config fgw.Config) {
	if !c.checkGatewayCodebase(gateway) {
		return
	}

	gatewayPath := utils.GatewayCodebasePath(gateway.Namespace, gateway.Name)

	jsonVersion, err := c.getVersion(gatewayPath, "config/version.json")
	if err != nil {
		return
	}

	log.Debug().Msgf("jsonVersion: %q, config version: %q", jsonVersion, config.GetVersion())

	if jsonVersion == config.GetVersion() {
		// config not changed, ignore updating
		log.Debug().Msgf("%s/config/version.json doesn't change, ignore updating...", gatewayPath)
		return
	}

	batch := repo.Batch{
		Basepath: gatewayPath,
		Items: []repo.BatchItem{
			{Path: "/config", Filename: "version.json", Content: fmt.Sprintf(`{"version": "%s"}`, config.GetVersion())},
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
				Path:     "/config/resources",
				Filename: strings.ToLower(resourceName(r)),
				Content:  toYAML(r),
			},
		)
	}

	for name, secret := range config.GetSecrets() {
		batch.Items = append(batch.Items,
			repo.BatchItem{
				Path:     "/config/secrets",
				Filename: name,
				Content:  secret,
			},
		)
	}

	for protocol, filters := range config.GetFilters() {
		for filterType, script := range filters {
			batch.Items = append(batch.Items,
				repo.BatchItem{
					Path:     fmt.Sprintf("/config/filters/%s", strings.ToLower(string(protocol))),
					Filename: fmt.Sprintf("%s.js", filterType),
					Content:  script,
				},
			)
		}
	}

	delItems, err := c.getDelItems(gatewayPath, batch)
	if err != nil {
		log.Error().Msgf("Get del items error: %s", err)
		return
	}
	batch.DelItems = delItems

	if err := c.repoClient.Batch([]repo.Batch{batch}); err != nil {
		log.Error().Msgf("Sync config of Gateway %s/%s to repo failed: %s", gateway.Namespace, gateway.Name, err)
		return
	}
}

func (c *GatewayProcessor) checkGatewayCodebase(gateway *gwv1.Gateway) bool {
	gatewayPath := utils.GatewayCodebasePath(gateway.Namespace, gateway.Name)
	parentPath := utils.GetDefaultGatewaysPath()

	if !c.repoClient.CodebaseExists(gatewayPath) {
		// Derive codebase only, don't commit it, the codebase will be committed when all configs are ready
		if err := c.repoClient.DeriveCodebaseOnly(gatewayPath, parentPath); err != nil {
			log.Error().Msgf("Failed to derive codebase %q: %s", gatewayPath, err)
			return false
		}
	}

	return true
}

func (c *GatewayProcessor) getDelItems(gatewayPath string, batch repo.Batch) ([]string, error) {
	files, err := c.repoClient.ListFiles(gatewayPath)
	if err != nil {
		log.Error().Msgf("List files in %q error: %s", gatewayPath, err)
		return nil, err
	}

	toDelete := sets.NewString(files...)

	for _, item := range batch.Items {
		toDelete.Delete(item.String())
	}

	return toDelete.UnsortedList(), nil
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
