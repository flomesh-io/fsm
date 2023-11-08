package retry

import (
	"k8s.io/utils/pointer"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// GetRetryConfigIfPortMatchesPolicy returns true if the port matches the retry policy
func GetRetryConfigIfPortMatchesPolicy(port int32, sessionStickyPolicy gwpav1alpha1.RetryPolicy) *gwpav1alpha1.RetryConfig {
	if len(sessionStickyPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range sessionStickyPolicy.Spec.Ports {
		if port == int32(policyPort.Port) {
			return ComputeRetryConfig(policyPort.Config, sessionStickyPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// ComputeRetryConfig computes the retry config based on the port config and default config
func ComputeRetryConfig(config *gwpav1alpha1.RetryConfig, defaultConfig *gwpav1alpha1.RetryConfig) *gwpav1alpha1.RetryConfig {
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

func mergeConfig(config *gwpav1alpha1.RetryConfig, defaultConfig *gwpav1alpha1.RetryConfig) *gwpav1alpha1.RetryConfig {
	cfgCopy := config.DeepCopy()

	if cfgCopy.NumRetries == nil {
		if defaultConfig.NumRetries != nil {
			cfgCopy.NumRetries = defaultConfig.NumRetries
		} else {
			cfgCopy.NumRetries = pointer.Int32(3)
		}
	}

	if cfgCopy.BackoffBaseInterval == nil {
		if defaultConfig.BackoffBaseInterval != nil {
			cfgCopy.BackoffBaseInterval = defaultConfig.BackoffBaseInterval
		} else {
			cfgCopy.BackoffBaseInterval = pointer.Int32(1)
		}
	}

	return cfgCopy
}

func setDefaultValues(config *gwpav1alpha1.RetryConfig) *gwpav1alpha1.RetryConfig {
	cfg := config.DeepCopy()

	if cfg.NumRetries == nil {
		cfg.NumRetries = pointer.Int32(3)
	}

	if cfg.BackoffBaseInterval == nil {
		cfg.BackoffBaseInterval = pointer.Int32(1)
	}

	return cfg
}
