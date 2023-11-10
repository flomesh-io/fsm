package gatewaytls

import (
	"k8s.io/utils/pointer"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// GetGatewayTLSConfigIfPortMatchesPolicy returns true if the port matches the access control policy
func GetGatewayTLSConfigIfPortMatchesPolicy(port gwv1beta1.PortNumber, gatewayTLSPolicy gwpav1alpha1.GatewayTLSPolicy) *gwpav1alpha1.GatewayTLSConfig {
	if len(gatewayTLSPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range gatewayTLSPolicy.Spec.Ports {
		if port == policyPort.Port {
			return ComputeGatewayTLSConfig(policyPort.Config, gatewayTLSPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// ComputeGatewayTLSConfig computes the access control config based on the config and default config
func ComputeGatewayTLSConfig(config *gwpav1alpha1.GatewayTLSConfig, defaultConfig *gwpav1alpha1.GatewayTLSConfig) *gwpav1alpha1.GatewayTLSConfig {
	switch {
	case config == nil && defaultConfig == nil:
		return nil
	case config == nil && defaultConfig != nil:
		return setDefaultValues(defaultConfig.DeepCopy())
	case config != nil && defaultConfig == nil:
		return setDefaultValues(config.DeepCopy())
	case config != nil && defaultConfig != nil:
		return mergeConfig(config, defaultConfig)
	}

	return nil
}

func mergeConfig(config *gwpav1alpha1.GatewayTLSConfig, defaultConfig *gwpav1alpha1.GatewayTLSConfig) *gwpav1alpha1.GatewayTLSConfig {
	cfgCopy := config.DeepCopy()

	if cfgCopy.MTLS == nil {
		if defaultConfig.MTLS != nil {
			// use port config
			cfgCopy.MTLS = defaultConfig.MTLS
		} else {
			// all nil, set to false
			cfgCopy.MTLS = pointer.Bool(false)
		}
	}

	return cfgCopy
}

func setDefaultValues(config *gwpav1alpha1.GatewayTLSConfig) *gwpav1alpha1.GatewayTLSConfig {
	cfg := config.DeepCopy()

	if cfg.MTLS == nil {
		cfg.MTLS = pointer.Bool(false)
	}

	return cfg
}
