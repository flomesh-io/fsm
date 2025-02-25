package k8s

import (
	"github.com/flomesh-io/fsm/pkg/connector"
)

type mapField struct {
	getter func(ins *ServiceInstance) string
	setter func(ins *ServiceInstance, value string) error
}

var (
	mapFields = map[string]*mapField{
		connector.CloudK8SPort: {
			getter: func(ins *ServiceInstance) string {
				return ins.FsmConnectorServicePort
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.FsmConnectorServicePort = value
				return nil
			},
		},
		connector.CloudK8SNS: {
			getter: func(ins *ServiceInstance) string {
				return ins.FsmConnectorServiceNamespace
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.FsmConnectorServiceNamespace = value
				return nil
			},
		},
		connector.ClusterSetKey: {
			getter: func(ins *ServiceInstance) string {
				return ins.FsmConnectorServiceClusterSet
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.FsmConnectorServiceClusterSet = value
				return nil
			},
		},
		connector.ConnectUIDKey: {
			getter: func(ins *ServiceInstance) string {
				return ins.FsmConnectorServiceConnectorUid
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.FsmConnectorServiceConnectorUid = value
				return nil
			},
		},
		connector.CloudHTTPViaGateway: {
			getter: func(ins *ServiceInstance) string {
				return ins.FsmConnectorServiceHTTPViaGateway
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.FsmConnectorServiceHTTPViaGateway = value
				return nil
			},
		},
		connector.CloudViaGatewayMode: {
			getter: func(ins *ServiceInstance) string {
				return ins.FsmConnectorServiceViaGatewayMode
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.FsmConnectorServiceViaGatewayMode = value
				return nil
			},
		},
	}
)
