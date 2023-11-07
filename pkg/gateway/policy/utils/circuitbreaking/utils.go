package circuitbreaking

import gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

// GetCircuitBreakingConfigIfPortMatchesPolicy returns true if the port matches the circuit breaking policy
func GetCircuitBreakingConfigIfPortMatchesPolicy(port int32, circuitBreakingPolicy gwpav1alpha1.CircuitBreakingPolicy) *gwpav1alpha1.CircuitBreakingConfig {
	if len(circuitBreakingPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range circuitBreakingPolicy.Spec.Ports {
		if port == int32(policyPort.Port) {
			return ComputeCircuitBreakingConfig(policyPort.Config, circuitBreakingPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// ComputeCircuitBreakingConfig computes the circuit breaking config based on the config and default config
func ComputeCircuitBreakingConfig(config *gwpav1alpha1.CircuitBreakingConfig, defaultConfig *gwpav1alpha1.CircuitBreakingConfig) *gwpav1alpha1.CircuitBreakingConfig {
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

func mergeConfig(config *gwpav1alpha1.CircuitBreakingConfig, defaultConfig *gwpav1alpha1.CircuitBreakingConfig) *gwpav1alpha1.CircuitBreakingConfig {
	cfgCopy := config.DeepCopy()

	if config.DegradedResponseContent == nil && defaultConfig.DegradedResponseContent != nil {
		cfgCopy.DegradedResponseContent = defaultConfig.DegradedResponseContent
	}

	if config.ErrorAmountThreshold == nil && defaultConfig.ErrorAmountThreshold != nil {
		cfgCopy.ErrorAmountThreshold = defaultConfig.ErrorAmountThreshold
	}

	if config.ErrorRatioThreshold == nil && defaultConfig.ErrorRatioThreshold != nil {
		cfgCopy.ErrorRatioThreshold = defaultConfig.ErrorRatioThreshold
	}

	if config.SlowAmountThreshold == nil && defaultConfig.SlowAmountThreshold != nil {
		cfgCopy.SlowAmountThreshold = defaultConfig.SlowAmountThreshold
	}

	if config.SlowRatioThreshold == nil && defaultConfig.SlowRatioThreshold != nil {
		cfgCopy.SlowRatioThreshold = defaultConfig.SlowRatioThreshold
	}

	if config.SlowTimeThreshold == nil && defaultConfig.SlowTimeThreshold != nil {
		cfgCopy.SlowTimeThreshold = defaultConfig.SlowTimeThreshold
	}

	return cfgCopy
}

func setDefaultValues(cfg *gwpav1alpha1.CircuitBreakingConfig) *gwpav1alpha1.CircuitBreakingConfig {
	cfg = cfg.DeepCopy()

	// do nothing for now

	return cfg
}
