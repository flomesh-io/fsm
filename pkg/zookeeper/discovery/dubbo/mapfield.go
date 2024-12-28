package dubbo

import (
	"fmt"
	"math"
	"strconv"

	"github.com/flomesh-io/fsm/pkg/connector"
)

type mapField struct {
	getter func(ins *ServiceInstance) string
	setter func(ins *ServiceInstance, value string) error
}

var (
	mapFields = map[string]*mapField{
		"interface": {
			getter: func(ins *ServiceInstance) string {
				return ins.Interface
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Interface = value
				return nil
			},
		},
		"methods": {
			getter: func(ins *ServiceInstance) string {
				return ins.Methods
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Methods = value
				return nil
			},
		},
		"application": {
			getter: func(ins *ServiceInstance) string {
				return ins.Application
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Application = value
				return nil
			},
		},
		"timestamp": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%d", ins.Timestamp)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if timestamp, err := strconv.ParseUint(value, 10, 64); err == nil {
					ins.Timestamp = timestamp
					return nil
				} else {
					return err
				}
			},
		},
		"pid": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%d", ins.PID)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if pid, err := strconv.Atoi(value); err == nil {
					if pid >= 0 && pid < math.MaxUint32 {
						ins.PID = uint32(pid)
					} else {
						ins.PID = 0
					}
					return nil
				} else {
					return err
				}
			},
		},
		"deprecated": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.Deprecated)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if deprecated, err := strconv.ParseBool(value); err == nil {
					ins.Deprecated = deprecated
					return nil
				} else {
					return err
				}
			},
		},
		"anyhost": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.Anyhost)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if anyhost, err := strconv.ParseBool(value); err == nil {
					ins.Anyhost = anyhost
					return nil
				} else {
					return err
				}
			},
		},
		"dynamic": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.Dynamic)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if dynamic, err := strconv.ParseBool(value); err == nil {
					ins.Dynamic = dynamic
					return nil
				} else {
					return err
				}
			},
		},
		"side": {
			getter: func(ins *ServiceInstance) string {
				return ins.Side
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Side = value
				return nil
			},
		},
		"version": {
			getter: func(ins *ServiceInstance) string {
				return ins.Version
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Version = value
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
		connector.CloudGRPCViaGateway: {
			getter: func(ins *ServiceInstance) string {
				return ins.FsmConnectorServiceGRPCViaGateway
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.FsmConnectorServiceGRPCViaGateway = value
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
