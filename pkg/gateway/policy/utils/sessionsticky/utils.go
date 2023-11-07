package sessionsticky

import (
	"k8s.io/utils/pointer"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// GetSessionStickyConfigIfPortMatchesPolicy returns true if the port matches the session sticky policy
func GetSessionStickyConfigIfPortMatchesPolicy(port int32, sessionStickyPolicy gwpav1alpha1.SessionStickyPolicy) *gwpav1alpha1.SessionStickyConfig {
	if len(sessionStickyPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range sessionStickyPolicy.Spec.Ports {
		if port == int32(policyPort.Port) {
			return getSessionStickyConfig(policyPort.Config, sessionStickyPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

func getSessionStickyConfig(config *gwpav1alpha1.SessionStickyConfig, defaultConfig *gwpav1alpha1.SessionStickyConfig) *gwpav1alpha1.SessionStickyConfig {
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

func mergeConfig(config *gwpav1alpha1.SessionStickyConfig, defaultConfig *gwpav1alpha1.SessionStickyConfig) *gwpav1alpha1.SessionStickyConfig {
	cfgCopy := config.DeepCopy()

	if cfgCopy.CookieName == nil {
		if defaultConfig.CookieName != nil {
			cfgCopy.CookieName = defaultConfig.CookieName
		} else {
			cfgCopy.CookieName = pointer.String("_srv_id")
		}
	}

	if cfgCopy.Expires == nil {
		if defaultConfig.Expires != nil {
			cfgCopy.Expires = defaultConfig.Expires
		} else {
			cfgCopy.Expires = pointer.Int32(3600)
		}
	}

	return cfgCopy
}

func setDefaultValues(config *gwpav1alpha1.SessionStickyConfig) *gwpav1alpha1.SessionStickyConfig {
	cfg := config.DeepCopy()

	if cfg.CookieName == nil {
		cfg.CookieName = pointer.String("_srv_id")
	}

	if cfg.Expires == nil {
		cfg.Expires = pointer.Int32(3600)
	}

	return cfg
}
