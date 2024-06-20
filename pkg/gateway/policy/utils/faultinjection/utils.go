package faultinjection

import (
	"reflect"
	"strings"

	"k8s.io/utils/pointer"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// GetFaultInjectionConfigIfRouteHostnameMatchesPolicy returns the fault injection config if the route hostname matches the policy
func GetFaultInjectionConfigIfRouteHostnameMatchesPolicy(routeHostname string, faultInjectionPolicy gwpav1alpha1.FaultInjectionPolicy) *gwpav1alpha1.FaultInjectionConfig {
	if len(faultInjectionPolicy.Spec.Hostnames) == 0 {
		return nil
	}

	for i := range faultInjectionPolicy.Spec.Hostnames {
		hostname := string(faultInjectionPolicy.Spec.Hostnames[i].Hostname)

		switch {
		case routeHostname == hostname:
			return ComputeFaultInjectionConfig(faultInjectionPolicy.Spec.Hostnames[i].Config, faultInjectionPolicy.Spec.DefaultConfig, faultInjectionPolicy.Spec.Unit)

		case strings.HasPrefix(routeHostname, "*"):
			if gwutils.HostnameMatchesWildcardHostname(hostname, routeHostname) {
				return ComputeFaultInjectionConfig(faultInjectionPolicy.Spec.Hostnames[i].Config, faultInjectionPolicy.Spec.DefaultConfig, faultInjectionPolicy.Spec.Unit)
			}

		case strings.HasPrefix(hostname, "*"):
			if gwutils.HostnameMatchesWildcardHostname(routeHostname, hostname) {
				return ComputeFaultInjectionConfig(faultInjectionPolicy.Spec.Hostnames[i].Config, faultInjectionPolicy.Spec.DefaultConfig, faultInjectionPolicy.Spec.Unit)
			}
		}
	}

	return nil
}

// GetFaultInjectionConfigIfHTTPRouteMatchesPolicy returns the fault injection config if the HTTP route matches the policy
func GetFaultInjectionConfigIfHTTPRouteMatchesPolicy(routeMatch gwv1.HTTPRouteMatch, faultInjectionPolicy gwpav1alpha1.FaultInjectionPolicy) *gwpav1alpha1.FaultInjectionConfig {
	if len(faultInjectionPolicy.Spec.HTTPFaultInjections) == 0 {
		return nil
	}

	for _, hr := range faultInjectionPolicy.Spec.HTTPFaultInjections {
		if reflect.DeepEqual(routeMatch, hr.Match) {
			return ComputeFaultInjectionConfig(hr.Config, faultInjectionPolicy.Spec.DefaultConfig, faultInjectionPolicy.Spec.Unit)
		}
	}

	return nil
}

// GetFaultInjectionConfigIfGRPCRouteMatchesPolicy returns the fault injection config if the GRPC route matches the policy
func GetFaultInjectionConfigIfGRPCRouteMatchesPolicy(routeMatch gwv1.GRPCRouteMatch, faultInjectionPolicy gwpav1alpha1.FaultInjectionPolicy) *gwpav1alpha1.FaultInjectionConfig {
	if len(faultInjectionPolicy.Spec.GRPCFaultInjections) == 0 {
		return nil
	}

	for _, gr := range faultInjectionPolicy.Spec.GRPCFaultInjections {
		if reflect.DeepEqual(routeMatch, gr.Match) {
			return ComputeFaultInjectionConfig(gr.Config, faultInjectionPolicy.Spec.DefaultConfig, faultInjectionPolicy.Spec.Unit)
		}
	}

	return nil
}

// ComputeFaultInjectionConfig computes the fault injection config based on the port config and default config
func ComputeFaultInjectionConfig(config *gwpav1alpha1.FaultInjectionConfig, defaultConfig *gwpav1alpha1.FaultInjectionConfig, unit *string) *gwpav1alpha1.FaultInjectionConfig {
	switch {
	case config == nil && defaultConfig == nil:
		return nil
	case config == nil && defaultConfig != nil:
		return setDefaultValues(defaultConfig.DeepCopy(), unit)
	case config != nil && defaultConfig == nil:
		return setDefaultValues(config.DeepCopy(), unit)
	case config != nil && defaultConfig != nil:
		return mergeConfig(config, defaultConfig, unit)
	}

	return nil
}

func mergeConfig(config *gwpav1alpha1.FaultInjectionConfig, defaultConfig *gwpav1alpha1.FaultInjectionConfig, unit *string) *gwpav1alpha1.FaultInjectionConfig {
	cfgCopy := config.DeepCopy()

	if hasValidDefaultDelay(cfgCopy, defaultConfig) {
		cfgCopy.Delay = defaultConfig.Delay
	}

	if hasValidDefaultAbort(cfgCopy, defaultConfig) {
		cfgCopy.Abort = defaultConfig.Abort
	}

	if cfgCopy.Delay != nil {
		if cfgCopy.Delay.Unit == nil {
			if defaultConfig.Delay != nil && defaultConfig.Delay.Unit != nil {
				cfgCopy.Delay.Unit = defaultConfig.Delay.Unit
			} else {
				if unit == nil {
					cfgCopy.Delay.Unit = pointer.String("ms")
				} else {
					cfgCopy.Delay.Unit = unit
				}
			}
		}

		if hasValidDefaultFixedDelay(cfgCopy, defaultConfig) {
			cfgCopy.Delay.Fixed = defaultConfig.Delay.Fixed
		}

		if hasValidDefaultRangeDelay(cfgCopy, defaultConfig) {
			cfgCopy.Delay.Range = defaultConfig.Delay.Range
		}
	}

	if cfgCopy.Abort != nil {
		if hasValidDefaultAbortStatusCode(cfgCopy, defaultConfig) {
			cfgCopy.Abort.StatusCode = defaultConfig.Abort.StatusCode
		}

		if hasValidDefaultAbortMessage(cfgCopy, defaultConfig) {
			cfgCopy.Abort.Message = defaultConfig.Abort.Message
		}
	}

	return cfgCopy
}

func hasValidDefaultDelay(cfgCopy *gwpav1alpha1.FaultInjectionConfig, defaultConfig *gwpav1alpha1.FaultInjectionConfig) bool {
	return cfgCopy.Delay == nil && cfgCopy.Abort == nil && defaultConfig.Delay != nil && defaultConfig.Abort == nil
}

func hasValidDefaultAbort(cfgCopy *gwpav1alpha1.FaultInjectionConfig, defaultConfig *gwpav1alpha1.FaultInjectionConfig) bool {
	return cfgCopy.Abort == nil && cfgCopy.Delay == nil && defaultConfig.Abort != nil && defaultConfig.Delay == nil
}

func hasValidDefaultAbortMessage(cfgCopy *gwpav1alpha1.FaultInjectionConfig, defaultConfig *gwpav1alpha1.FaultInjectionConfig) bool {
	return cfgCopy.Abort.Message == nil &&
		defaultConfig.Delay == nil &&
		defaultConfig.Abort != nil &&
		defaultConfig.Abort.Message != nil
}

func hasValidDefaultAbortStatusCode(cfgCopy *gwpav1alpha1.FaultInjectionConfig, defaultConfig *gwpav1alpha1.FaultInjectionConfig) bool {
	return cfgCopy.Abort.StatusCode == nil &&
		defaultConfig.Delay == nil &&
		defaultConfig.Abort != nil &&
		defaultConfig.Abort.StatusCode != nil
}

func hasValidDefaultRangeDelay(cfgCopy *gwpav1alpha1.FaultInjectionConfig, defaultConfig *gwpav1alpha1.FaultInjectionConfig) bool {
	return cfgCopy.Delay.Range == nil &&
		cfgCopy.Delay.Fixed == nil &&
		defaultConfig.Abort == nil &&
		defaultConfig.Delay != nil &&
		defaultConfig.Delay.Range != nil &&
		defaultConfig.Delay.Fixed == nil
}

func hasValidDefaultFixedDelay(cfgCopy *gwpav1alpha1.FaultInjectionConfig, defaultConfig *gwpav1alpha1.FaultInjectionConfig) bool {
	return cfgCopy.Delay.Fixed == nil &&
		cfgCopy.Delay.Range == nil &&
		defaultConfig.Abort == nil &&
		defaultConfig.Delay != nil &&
		defaultConfig.Delay.Fixed != nil &&
		defaultConfig.Delay.Range == nil
}

func setDefaultValues(config *gwpav1alpha1.FaultInjectionConfig, unit *string) *gwpav1alpha1.FaultInjectionConfig {
	cfg := config.DeepCopy()

	if cfg.Delay != nil && cfg.Delay.Unit == nil {
		if unit == nil {
			cfg.Delay.Unit = pointer.String("ms")
		} else {
			cfg.Delay.Unit = unit
		}
	}

	return cfg
}
