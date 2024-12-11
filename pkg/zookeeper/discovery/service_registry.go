package discovery

import (
	"path"

	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/pkg/errors"
)

// registerService register service to zookeeper
func (sd *ServiceDiscovery) registerService(instance ServiceInstance) error {
	categoryServiceName := path.Join(instance.ServiceName(), sd.category)
	instancePath := sd.ops.PathForInstance(sd.basePath, categoryServiceName, instance.InstanceId())
	data, err := instance.Marshal()
	if err != nil {
		return err
	}

	if err = sd.client.Delete(instancePath); err != nil {
		log.Info().Msgf("Failed when trying to delete node %s, will continue with the registration process. "+
			"This is designed to avoid previous ephemeral node hold the position,"+
			" so it's normal for this action to fail because the node might not exist or has been deleted, error msg is %s.", instancePath, err.Error())
	}

	if err = sd.client.CreateTempWithValue(instancePath, data); errors.Is(err, zk.ErrNodeExists) {
		_, state, _ := sd.client.GetContent(instancePath)
		if state != nil {
			_, err = sd.client.SetContent(instancePath, data, state.Version+1)
			if err != nil {
				log.Debug().Msgf("Try to update the node data failed. In most cases, it's not a problem. ")
			}
		}
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

// RegisterService register service to zookeeper, and ensure cache is consistent with zookeeper
func (sd *ServiceDiscovery) RegisterService(instance ServiceInstance) error {
	value, _ := sd.services.LoadOrStore(instance.InstanceId(), &serviceEntry{})
	entry, ok := value.(*serviceEntry)
	if !ok {
		return errors.New("[ServiceDiscovery] services value not serviceEntry")
	}

	entry.Lock()
	defer entry.Unlock()

	entry.instance = instance
	if err := sd.registerService(instance); err != nil {
		return err
	}
	return nil
}

// UnregisterService un-register service in zookeeper and delete service in cache
func (sd *ServiceDiscovery) UnregisterService(instance ServiceInstance) error {
	if _, ok := sd.services.Load(instance.InstanceId()); !ok {
		return nil
	}
	sd.services.Delete(instance.InstanceId())
	return sd.unregisterService(instance)
}

// unregisterService un-register service in zookeeper
func (sd *ServiceDiscovery) unregisterService(instance ServiceInstance) error {
	serviceName := sd.ops.KtoCName(instance.ServiceName())
	if len(serviceName) == 0 {
		return nil
	}
	categoryServiceName := path.Join(serviceName, sd.category)
	instancePath := sd.ops.PathForInstance(sd.basePath, categoryServiceName, instance.InstanceId())
	return sd.client.Delete(instancePath)
}
