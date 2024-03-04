package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	connectorv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
)

func (c *client) Refresh() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if spec, uid, ok := c.GetConnector(); ok {
		if hash, err := hashstructure.Hash(spec, hashstructure.FormatV2, nil); err == nil {
			if c.connectorHash == hash {
				return
			}

			if err = c.validateConnectorSpec(spec); err != nil {
				log.Fatal().Err(err).Msg("Error validating connector spec parameters")
			}

			c.connectorSpec = spec
			c.connectorUID = uid
			c.connectorHash = hash

			fmt.Println(uid, hash)
			bytes, _ := json.MarshalIndent(spec, "", " ")
			fmt.Println(string(bytes))

			if len(c.cancelFuncs) > 0 {
				for _, cancelFunc := range c.cancelFuncs {
					cancelFunc()
				}
				c.cancelFuncs = nil
			}

			ctx, cancelFunc := context.WithCancel(context.Background())
			c.cancelFuncs = append(c.cancelFuncs, cancelFunc)
			go c.startSync(ctx)
		}
	}
}

func (c *client) startSync(ctx context.Context) {
	var err error
	var discClient provider.ServiceDiscoveryClient = nil
	if consulSpec, consulOk := c.connectorSpec.(connectorv1alpha1.ConsulSpec); consulOk {
		c.initConsulConnectorConfig(consulSpec)

		appendTagSet := ToSet(Cfg.K2C.FlagAppendTags)
		discClient, err = provider.GetConsulDiscoveryClient(Cfg.HttpAddr,
			Cfg.AsInternalServices, Cfg.C2K.FlagClusterId,
			appendTagSet)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		}
	} else if eurekaSpec, eurekaOk := c.connectorSpec.(connectorv1alpha1.EurekaSpec); eurekaOk {
		c.initEurekaConnectorConfig(eurekaSpec)

		appendMetadataKeySet := ToSet(Cfg.K2C.FlagAppendMetadataKeys)
		appendMetadataValueSet := ToSet(Cfg.K2C.FlagAppendMetadataValues)
		discClient, err = provider.GetEurekaDiscoveryClient(Cfg.HttpAddr,
			Cfg.AsInternalServices, Cfg.C2K.FlagClusterId,
			appendMetadataKeySet, appendMetadataValueSet)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		}
	} else if nacosSpec, nacosOk := c.connectorSpec.(connectorv1alpha1.NacosSpec); nacosOk {
		c.initNacosConnectorConfig(nacosSpec)

		appendMetadataKeySet := ToSet(Cfg.K2C.FlagAppendMetadataKeys)
		appendMetadataValueSet := ToSet(Cfg.K2C.FlagAppendMetadataValues)
		discClient, err = provider.GetNacosDiscoveryClient(Cfg.HttpAddr,
			Cfg.Nacos.FlagUsername, Cfg.Nacos.FlagPassword, Cfg.Nacos.FlagNamespaceId, Cfg.C2K.FlagClusterId,
			Cfg.K2C.Nacos.FlagClusterId, Cfg.K2C.Nacos.FlagGroupId, Cfg.C2K.Nacos.FlagClusterSet, Cfg.C2K.Nacos.FlagGroupSet,
			Cfg.AsInternalServices, appendMetadataKeySet, appendMetadataValueSet)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		}
	} else if machineSpec, machineOk := c.connectorSpec.(connectorv1alpha1.MachineSpec); machineOk {
		c.initMachineConnectorConfig(machineSpec)

		discClient, err = provider.GetMachineDiscoveryClient(c.machineClient, Cfg.DeriveNamespace, Cfg.AsInternalServices, Cfg.C2K.FlagClusterId)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		}
	} else if gatewaySpec, gatewayOk := c.connectorSpec.(connectorv1alpha1.GatewaySpec); gatewayOk {
		c.initGatewayConnectorConfig(gatewaySpec)
	}

	if Cfg.SyncCloudToK8s {
		go syncCtoK(ctx, c.kubeClient, c.configClient, discClient)
	}

	if Cfg.SyncK8sToCloud {
		go syncKtoC(ctx, c.kubeClient, c.configClient, discClient)
	}

	if Cfg.SyncK8sToGateway {
		gatewayClient := gwapi.NewForConfigOrDie(c.kubeConfig)
		go syncKtoG(ctx, c.kubeClient, c.configClient, gatewayClient)
	}
}
