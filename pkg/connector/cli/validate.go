package cli

import (
	"fmt"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
)

func (c *client) validateConnectorSpec(spec interface{}) error {
	if consulSpec, ok := spec.(ctv1.ConsulSpec); ok {
		if len(consulSpec.HTTPAddr) == 0 {
			return fmt.Errorf("please specify service discovery and registration server address")
		}
		if consulSpec.SyncFromK8S.Enable || consulSpec.SyncToK8S.Enable {
			if len(consulSpec.DeriveNamespace) == 0 {
				return fmt.Errorf("please specify the cloud derive namespace")
			}
		}
	}

	if eurekaSpec, ok := spec.(ctv1.EurekaSpec); ok {
		if len(eurekaSpec.HTTPAddr) == 0 {
			return fmt.Errorf("please specify service discovery and registration server address")
		}
		if eurekaSpec.SyncFromK8S.Enable || eurekaSpec.SyncToK8S.Enable {
			if len(eurekaSpec.DeriveNamespace) == 0 {
				return fmt.Errorf("please specify the cloud derive namespace")
			}
		}
	}

	if nacosSpec, ok := spec.(ctv1.NacosSpec); ok {
		if len(nacosSpec.HTTPAddr) == 0 {
			return fmt.Errorf("please specify service discovery and registration server address")
		}
		if nacosSpec.SyncFromK8S.Enable || nacosSpec.SyncToK8S.Enable {
			if len(nacosSpec.DeriveNamespace) == 0 {
				return fmt.Errorf("please specify the cloud derive namespace")
			}
		}
		if nacosSpec.SyncFromK8S.Enable {
			if len(nacosSpec.SyncFromK8S.ClusterId) == 0 {
				return fmt.Errorf("please specify the nacos cluster id")
			}
			if len(nacosSpec.SyncFromK8S.GroupId) == 0 {
				return fmt.Errorf("please specify the nacos group id using")
			}
		}
	}

	if zookeeperSpec, ok := spec.(ctv1.ZookeeperSpec); ok {
		if len(zookeeperSpec.HTTPAddr) == 0 {
			return fmt.Errorf("please specify service discovery and registration server address")
		}
		if zookeeperSpec.SyncFromK8S.Enable || zookeeperSpec.SyncToK8S.Enable {
			if len(zookeeperSpec.DeriveNamespace) == 0 {
				return fmt.Errorf("please specify the cloud derive namespace")
			}
			if len(zookeeperSpec.BasePath) == 0 {
				return fmt.Errorf("please specify the zookeeper base path")
			}
			if len(zookeeperSpec.Category) == 0 {
				return fmt.Errorf("please specify the zookeeper category")
			}
			if len(zookeeperSpec.Adaptor) == 0 {
				return fmt.Errorf("please specify the zookeeper adaptor")
			}
		}
	}

	return nil
}
