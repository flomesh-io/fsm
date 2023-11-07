package healthcheck

import gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

// GetHealthCheckConfigIfPortMatchesPolicy returns true if the port matches the circuit breaking policy
func GetHealthCheckConfigIfPortMatchesPolicy(port int32, healthCheckPolicy gwpav1alpha1.HealthCheckPolicy) *gwpav1alpha1.HealthCheckConfig {
	if len(healthCheckPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range healthCheckPolicy.Spec.Ports {
		if port == int32(policyPort.Port) {
			return ComputeHealthCheckConfig(policyPort.Config, healthCheckPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// ComputeHealthCheckConfig computes the circuit breaking config based on the port config and default config
func ComputeHealthCheckConfig(config *gwpav1alpha1.HealthCheckConfig, defaultConfig *gwpav1alpha1.HealthCheckConfig) *gwpav1alpha1.HealthCheckConfig {
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

func mergeConfig(config *gwpav1alpha1.HealthCheckConfig, defaultConfig *gwpav1alpha1.HealthCheckConfig) *gwpav1alpha1.HealthCheckConfig {
	cfgCopy := config.DeepCopy()

	if cfgCopy.Path == nil && defaultConfig.Path != nil {
		cfgCopy.Path = defaultConfig.Path
	}

	if len(cfgCopy.Matches) == 0 && len(defaultConfig.Matches) > 0 {
		cfgCopy.Matches = make([]gwpav1alpha1.HealthCheckMatch, 0)
		cfgCopy.Matches = append(cfgCopy.Matches, defaultConfig.Matches...)
	}

	if cfgCopy.FailTimeout == nil && defaultConfig.FailTimeout != nil {
		cfgCopy.FailTimeout = defaultConfig.FailTimeout
	}

	return cfgCopy
}

func setDefaultValues(config *gwpav1alpha1.HealthCheckConfig) *gwpav1alpha1.HealthCheckConfig {
	cfg := config.DeepCopy()

	if cfg.Path != nil && len(cfg.Matches) == 0 {
		cfg.Matches = []gwpav1alpha1.HealthCheckMatch{
			{
				StatusCodes: []int32{200},
			},
		}
	}

	return cfg
}
