package repo

import (
	"fmt"
	"sort"
	"strings"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/client"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

func (s *Server) updatePlugins() (pluginSet mapset.Set, pluginPri map[string]float32) {
	var pluginItems []client.BatchItem
	var pluginVers []string
	pluginSet = mapset.NewSet()
	pluginPri = make(map[string]float32)

	plugins := s.catalog.GetPlugins()
	for _, pluginItem := range plugins {
		uri := getPluginURI(pluginItem.Name)
		bytes := []byte(pluginItem.Script)
		pluginSet.Add(pluginItem.Name)
		pluginPri[pluginItem.Name] = pluginItem.Priority
		pluginItems = append(pluginItems, client.BatchItem{
			Filename: uri,
			Content:  bytes,
		})
		pluginVers = append(pluginVers, fmt.Sprintf("%s:%f:%d", uri, pluginItem.Priority, Hash(bytes)))
	}

	diffSet := s.pluginSet.Difference(pluginSet)
	diffPlugins := diffSet.ToSlice()
	for _, pluginName := range diffPlugins {
		pluginItems = append(pluginItems, client.BatchItem{
			Filename: getPluginURI(pluginName.(string)),
			Obsolete: true,
		})
	}
	if len(pluginItems) > 0 {
		sort.Strings(pluginVers)
		pluginSetHash := Hash([]byte(strings.Join(pluginVers, "")))
		pluginSetVersion := fmt.Sprintf("%d", pluginSetHash)

		s.pluginMutex.Lock()
		defer s.pluginMutex.Unlock()

		if s.pluginSetVersion == pluginSetVersion {
			return
		}
		_, err := s.repoClient.Batch(pluginSetVersion, []client.Batch{
			{
				Basepath: fsmCodebase,
				Items:    pluginItems,
			},
		})
		if err != nil {
			log.Error().Err(err)
		} else {
			s.pluginSet = pluginSet
			s.pluginPri = pluginPri
			s.pluginSetVersion = pluginSetVersion
		}
	}
	return
}

// getPluginURI return the URI of the plugin.
func getPluginURI(name string) string {
	return fmt.Sprintf("plugins/%s.js", name)
}

func matchPluginChain(pluginChain *trafficpolicy.PluginChain, ns *corev1.Namespace, labelMap map[string]string) bool {
	matchedNamespace := false
	matchedPod := false

	if pluginChain.Selectors.NamespaceSelector != nil {
		labelSelector, errSelector := metav1.LabelSelectorAsSelector(pluginChain.Selectors.NamespaceSelector)
		if errSelector == nil {
			matchedNamespace = labelSelector.Matches(labels.Set(ns.GetLabels()))
		} else {
			log.Err(errSelector).Str("namespace", pluginChain.Namespace).Str("PluginChan", pluginChain.Name)
			return false
		}
	} else {
		matchedNamespace = true
	}

	if pluginChain.Selectors.PodSelector != nil {
		labelSelector, errSelector := metav1.LabelSelectorAsSelector(pluginChain.Selectors.PodSelector)
		if errSelector == nil {
			matchedPod = labelSelector.Matches(labels.Set(labelMap))
		} else {
			log.Err(errSelector).Str("namespace", pluginChain.Namespace).Str("PluginChan", pluginChain.Name)
			return false
		}
	} else {
		matchedPod = true
	}

	return matchedNamespace && matchedPod
}

func walkPluginChain(pluginChains []*trafficpolicy.PluginChain, ns *corev1.Namespace, labelMap map[string]string, pluginSet mapset.Set, s *Server, proxy *pipy.Proxy) (plugin2MountPoints map[string]*map[string]*runtime.RawExtension, mountPoint2Plugins map[string]mapset.Set) {
	plugin2MountPoints = make(map[string]*map[string]*runtime.RawExtension)
	mountPoint2Plugins = make(map[string]mapset.Set)

	for _, pluginChain := range pluginChains {
		matched := matchPluginChain(pluginChain, ns, labelMap)
		if !matched {
			continue
		}
		for _, chain := range pluginChain.Chains {
			for _, pluginName := range chain.Plugins {
				if !pluginSet.Contains(pluginName) {
					if len(s.pluginSetVersion) > 0 {
						log.Warn().Str("proxy", proxy.String()).
							Str("plugin", pluginName).
							Msg("Could not find plugin for connecting proxy.")
					}
					if s.retryProxiesJob != nil {
						s.retryProxiesJob()
					}
					continue
				}

				mountPointSet, existPointSet := plugin2MountPoints[pluginName]
				if !existPointSet {
					mountPointMap := make(map[string]*runtime.RawExtension)
					mountPointSet = &mountPointMap
					plugin2MountPoints[pluginName] = mountPointSet
				}
				if _, exist := (*mountPointSet)[chain.Name]; !exist {
					(*mountPointSet)[chain.Name] = nil
				}

				mountedPluginSet, existPluginSet := mountPoint2Plugins[chain.Name]
				if !existPluginSet {
					mountedPluginSet = mapset.NewSet()
					mountPoint2Plugins[chain.Name] = mountedPluginSet
				}
				if !mountedPluginSet.Contains(pluginName) {
					mountedPluginSet.Add(pluginName)
				}
			}
		}
	}
	return
}

func walkPluginConfig(cataloger catalog.MeshCataloger, plugin2MountPoint2Config map[string]*map[string]*runtime.RawExtension) map[string]map[string]*map[string]*runtime.RawExtension {
	meshSvc2Plugin2MountPoint2Config := make(map[string]map[string]*map[string]*runtime.RawExtension)
	pluginConfigs := cataloger.GetPluginConfigs()
	if len(pluginConfigs) > 0 {
		for _, pluginConfig := range pluginConfigs {
			mountPoint2ConfigItem, existMountPoint2Config := plugin2MountPoint2Config[pluginConfig.Plugin]
			if !existMountPoint2Config {
				continue
			}
			for mountPoint := range *mountPoint2ConfigItem {
				(*mountPoint2ConfigItem)[mountPoint] = &pluginConfig.Config // #nosec G601
			}
			for _, destinationRef := range pluginConfig.DestinationRefs {
				if destinationRef.Kind == policyv1alpha1.KindService {
					meshSvc := service.MeshService{
						Namespace: destinationRef.Namespace,
						Name:      destinationRef.Name,
					}
					plugin2MountPoint2ConfigItem, exist := meshSvc2Plugin2MountPoint2Config[meshSvc.String()]
					if !exist {
						plugin2MountPoint2ConfigItem = make(map[string]*map[string]*runtime.RawExtension)
						meshSvc2Plugin2MountPoint2Config[meshSvc.String()] = plugin2MountPoint2ConfigItem
					}
					plugin2MountPoint2ConfigItem[pluginConfig.Plugin] = mountPoint2ConfigItem
				}
			}
		}
	}
	return meshSvc2Plugin2MountPoint2Config
}

func setSidecarChain(cfg configurator.Configurator, pipyConf *PipyConf, pluginPri map[string]float32, mountPoint2Plugins map[string]mapset.Set) {
	pluginChains := cfg.GetGlobalPluginChains()
	if len(pluginPri) > 0 && len(mountPoint2Plugins) > 0 {
		for mountPoint, plugins := range mountPoint2Plugins {
			pluginItems := pluginChains[mountPoint]
			pluginSlice := plugins.ToSlice()
			for _, item := range pluginSlice {
				if pri, exist := pluginPri[item.(string)]; exist {
					pluginItems = append(pluginItems, trafficpolicy.Plugin{
						Name:     item.(string),
						Priority: pri,
					})
				}
			}
			pluginChains[mountPoint] = pluginItems
		}
	}

	pipyConf.Chains = make(map[string][]string)
	for mountPoint, pluginItems := range pluginChains {
		pluginSlice := PluginSlice(pluginItems)
		if len(pluginSlice) > 0 {
			var pluginURIs []string
			sort.Sort(&pluginSlice)
			for _, pluginItem := range pluginItems {
				if pluginItem.BuildIn {
					pluginURIs = append(pluginURIs, fmt.Sprintf("%s.js", pluginItem.Name))
				} else {
					pluginURIs = append(pluginURIs, getPluginURI(pluginItem.Name))
				}
			}
			pipyConf.Chains[mountPoint] = pluginURIs
		}
	}
}

func (p *PipyConf) getTrafficMatchPluginConfigs(trafficMatch string) map[string]*runtime.RawExtension {
	segs := strings.Split(trafficMatch, "_")
	meshSvc := segs[1]

	direct := segs[0]
	switch segs[0] {
	case "ingress":
	case "acl":
	case "exp":
		direct = "inbound"
	}
	mountPoint := fmt.Sprintf("%s-%s", direct, segs[3])

	plugin2MountPoint2Config, exist := p.pluginPolicies[meshSvc]
	if !exist {
		return nil
	}
	var pluginConfigs map[string]*runtime.RawExtension
	for pluginName, mountPoint2Config := range plugin2MountPoint2Config {
		if configLoop, existConfig := (*mountPoint2Config)[mountPoint]; existConfig {
			config := configLoop
			if pluginConfigs == nil {
				pluginConfigs = make(map[string]*runtime.RawExtension)
			}
			pluginConfigs[pluginName] = config
		}
	}
	return pluginConfigs
}
