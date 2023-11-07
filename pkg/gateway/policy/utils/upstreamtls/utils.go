package upstreamtls

import (
	"k8s.io/utils/pointer"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// GetUpstreamTLSConfigIfPortMatchesPolicy returns the upstream TLS config if the port matches the policy
func GetUpstreamTLSConfigIfPortMatchesPolicy(port int32, upstreamTLSPolicy gwpav1alpha1.UpstreamTLSPolicy) *gwpav1alpha1.UpstreamTLSConfig {
	if len(upstreamTLSPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range upstreamTLSPolicy.Spec.Ports {
		if port == int32(policyPort.Port) {
			return getUpstreamTLSConfig(policyPort.Config, upstreamTLSPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

func getUpstreamTLSConfig(config *gwpav1alpha1.UpstreamTLSConfig, defaultConfig *gwpav1alpha1.UpstreamTLSConfig) *gwpav1alpha1.UpstreamTLSConfig {
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

func mergeConfig(config *gwpav1alpha1.UpstreamTLSConfig, defaultConfig *gwpav1alpha1.UpstreamTLSConfig) *gwpav1alpha1.UpstreamTLSConfig {
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

func setDefaultValues(config *gwpav1alpha1.UpstreamTLSConfig) *gwpav1alpha1.UpstreamTLSConfig {
	cfg := config.DeepCopy()

	if cfg.MTLS == nil {
		cfg.MTLS = pointer.Bool(false)
	}

	return cfg
}
