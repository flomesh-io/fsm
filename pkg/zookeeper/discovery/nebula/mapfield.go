package nebula

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
		"project": {
			getter: func(ins *ServiceInstance) string {
				return ins.Project
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Project = value
				return nil
			},
		},
		"owner": {
			getter: func(ins *ServiceInstance) string {
				return ins.Owner
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Owner = value
				return nil
			},
		},
		"ops": {
			getter: func(ins *ServiceInstance) string {
				return ins.Ops
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Ops = value
				return nil
			},
		},
		"category": {
			getter: func(ins *ServiceInstance) string {
				return ins.Category
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.Category = value
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
		"grpc": {
			getter: func(ins *ServiceInstance) string {
				return ins.GRPC
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.GRPC = value
				return nil
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
		"group": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.Group)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if group, err := strconv.ParseBool(value); err == nil {
					ins.Group = group
					return nil
				} else {
					return err
				}
			},
		},
		"weight": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%d", ins.Weight)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if weight, err := strconv.Atoi(value); err == nil {
					if weight >= 0 && weight < math.MaxUint32 {
						ins.Weight = uint32(weight)
					} else {
						ins.Weight = 0
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
		"master": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.Master)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if master, err := strconv.ParseBool(value); err == nil {
					ins.Master = master
					return nil
				} else {
					return err
				}
			},
		},
		"default.async": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.DefaultAsync)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if async, err := strconv.ParseBool(value); err == nil {
					ins.DefaultAsync = async
					return nil
				} else {
					return err
				}
			},
		},
		"default.cluster": {
			getter: func(ins *ServiceInstance) string {
				return ins.DefaultCluster
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.DefaultCluster = value
				return nil
			},
		},
		"default.connections": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%d", ins.DefaultConnections)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if connections, err := strconv.Atoi(value); err == nil {
					if connections >= 0 && connections < math.MaxUint32 {
						ins.DefaultConnections = uint32(connections)
					} else {
						ins.DefaultConnections = 0
					}
					return nil
				} else {
					return err
				}
			},
		},
		"default.loadbalance": {
			getter: func(ins *ServiceInstance) string {
				return ins.DefaultLoadBalance
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.DefaultLoadBalance = value
				return nil
			},
		},
		"default.requests": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%d", ins.DefaultRequests)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if requests, err := strconv.Atoi(value); err == nil {
					if requests >= 0 && requests < math.MaxUint32 {
						ins.DefaultRequests = uint32(requests)
					} else {
						ins.DefaultRequests = 0
					}
					return nil
				} else {
					return err
				}
			},
		},
		"default.reties": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%d", ins.DefaultReties)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if reties, err := strconv.Atoi(value); err == nil {
					if reties >= 0 && reties < math.MaxUint32 {
						ins.DefaultReties = uint32(reties)
					} else {
						ins.DefaultReties = 0
					}
					return nil
				} else {
					return err
				}
			},
		},
		"default.timeout": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%d", ins.DefaultTimeout)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if timeout, err := strconv.Atoi(value); err == nil {
					if timeout >= 0 && timeout < math.MaxUint32 {
						ins.DefaultTimeout = uint32(timeout)
					} else {
						ins.DefaultTimeout = 0
					}
					return nil
				} else {
					return err
				}
			},
		},
		"service.type": {
			getter: func(ins *ServiceInstance) string {
				return ins.ServiceType
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.ServiceType = value
				return nil
			},
		},
		"real.ip": {
			getter: func(ins *ServiceInstance) string {
				return ins.RealIP
			},
			setter: func(ins *ServiceInstance, value string) error {
				ins.RealIP = value
				return nil
			},
		},
		"real.port": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%d", ins.RealPort)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if port, err := strconv.Atoi(value); err == nil {
					if port >= 0 && port < math.MaxUint16 {
						ins.RealPort = uint16(port)
					} else {
						ins.RealPort = 0
					}
					return nil
				} else {
					return err
				}
			},
		},
		"access.protected": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.AccessProtected)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if protected, err := strconv.ParseBool(value); err == nil {
					ins.AccessProtected = protected
					return nil
				} else {
					return err
				}
			},
		},
		"accesslog": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.Accesslog)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if accesslog, err := strconv.ParseBool(value); err == nil {
					ins.Accesslog = accesslog
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
		"token": {
			getter: func(ins *ServiceInstance) string {
				return fmt.Sprintf("%t", ins.Token)
			},
			setter: func(ins *ServiceInstance, value string) error {
				if token, err := strconv.ParseBool(value); err == nil {
					ins.Token = token
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
