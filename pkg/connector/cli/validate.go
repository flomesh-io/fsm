package cli

import (
	"fmt"

	connectorv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
)

func (c *client) validateConnectorSpec(spec interface{}) error {
	if consulSpec, ok := spec.(connectorv1alpha1.ConsulSpec); ok {
		if len(consulSpec.HTTPAddr) == 0 {
			return fmt.Errorf("please specify service discovery and registration server address")
		}
		if consulSpec.SyncFromK8S.Enable || consulSpec.SyncToK8S.Enable {
			if len(consulSpec.DeriveNamespace) == 0 {
				return fmt.Errorf("please specify the cloud derive namespace")
			}
		}
	}

	if eurekaSpec, ok := spec.(connectorv1alpha1.EurekaSpec); ok {
		if len(eurekaSpec.HTTPAddr) == 0 {
			return fmt.Errorf("please specify service discovery and registration server address")
		}
		if eurekaSpec.SyncFromK8S.Enable || eurekaSpec.SyncToK8S.Enable {
			if len(eurekaSpec.DeriveNamespace) == 0 {
				return fmt.Errorf("please specify the cloud derive namespace")
			}
		}
	}

	if nacosSpec, ok := spec.(connectorv1alpha1.NacosSpec); ok {
		if len(nacosSpec.HTTPAddr) == 0 {
			return fmt.Errorf("please specify service discovery and registration server address")
		}
		if nacosSpec.SyncFromK8S.Enable || nacosSpec.SyncToK8S.Enable {
			if len(nacosSpec.DeriveNamespace) == 0 {
				return fmt.Errorf("please specify the cloud derive namespace")
			}
			if len(nacosSpec.NamespaceId) == 0 {
				return fmt.Errorf("please specify the nacos namespace id using")
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
	return nil
}
