package cli

import (
	"github.com/mitchellh/hashstructure/v2"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
)

func (c *client) Refresh() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, spec, uid, ok := c.GetConnector(); ok {
		if hash, err := hashstructure.Hash(spec, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			}); err == nil {
			if c.connectorHash == hash {
				return
			}

			if err = c.validateConnectorSpec(spec); err != nil {
				log.Fatal().Err(err).Msg("Error validating connector spec parameters")
			}

			c.connectorSpec = spec
			c.connectorUID = uid
			c.connectorHash = hash

			if len(c.cancelFuncs) > 0 {
				for _, cancelFunc := range c.cancelFuncs {
					cancelFunc()
				}
				c.cancelFuncs = nil
			}

			go c.startSync()
		}
	}
}

func (c *client) startSync() {
	var err error
	if consulSpec, consulOk := c.connectorSpec.(ctv1.ConsulSpec); consulOk {
		c.initConsulConnectorConfig(consulSpec)

		c.discClient, err = provider.GetConsulDiscoveryClient(c)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		} else {
			c.cancelFuncs = append(c.cancelFuncs, c.discClient.Close)
		}
	} else if eurekaSpec, eurekaOk := c.connectorSpec.(ctv1.EurekaSpec); eurekaOk {
		c.initEurekaConnectorConfig(eurekaSpec)

		c.discClient, err = provider.GetEurekaDiscoveryClient(c)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		} else {
			c.cancelFuncs = append(c.cancelFuncs, c.discClient.Close)
		}
	} else if nacosSpec, nacosOk := c.connectorSpec.(ctv1.NacosSpec); nacosOk {
		c.initNacosConnectorConfig(nacosSpec)

		c.discClient, err = provider.GetNacosDiscoveryClient(c)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		} else {
			c.cancelFuncs = append(c.cancelFuncs, c.discClient.Close)
		}
	} else if zookeeperSpec, zookeeperOk := c.connectorSpec.(ctv1.ZookeeperSpec); zookeeperOk {
		c.initZookeeperConnectorConfig(zookeeperSpec)

		c.discClient, err = provider.GetZookeeperDiscoveryClient(c)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		} else {
			c.cancelFuncs = append(c.cancelFuncs, c.discClient.Close)
		}
	} else if machineSpec, machineOk := c.connectorSpec.(ctv1.MachineSpec); machineOk {
		c.initMachineConnectorConfig(machineSpec)

		c.discClient, err = provider.GetMachineDiscoveryClient(c, c.machineClient)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating service discovery and registration client")
			log.Fatal().Msg("Error creating service discovery and registration client")
		} else {
			c.cancelFuncs = append(c.cancelFuncs, c.discClient.Close)
		}
	} else if gatewaySpec, gatewayOk := c.connectorSpec.(ctv1.GatewaySpec); gatewayOk {
		c.initGatewayConnectorConfig(gatewaySpec)
	}

	if c.SyncCloudToK8s() {
		go c.syncCtoK()
	}

	if c.SyncK8sToCloud() {
		go c.syncKtoC()
	}

	if c.SyncK8sToGateway() {
		go c.syncKtoG()
	}
}

func (c *client) WaitLimiter() {
	_ = c.limiter.Wait(c.context)
}
