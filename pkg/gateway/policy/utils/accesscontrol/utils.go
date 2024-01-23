package accesscontrol

import (
	"reflect"
	"strings"

	"k8s.io/utils/pointer"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// GetAccessControlConfigIfPortMatchesPolicy returns true if the port matches the access control policy
func GetAccessControlConfigIfPortMatchesPolicy(port gwv1.PortNumber, accessControlPolicy gwpav1alpha1.AccessControlPolicy) *gwpav1alpha1.AccessControlConfig {
	if len(accessControlPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range accessControlPolicy.Spec.Ports {
		if port == policyPort.Port {
			return ComputeAccessControlConfig(policyPort.Config, accessControlPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// GetAccessControlConfigIfRouteHostnameMatchesPolicy returns the access control config if the route hostname matches the policy
func GetAccessControlConfigIfRouteHostnameMatchesPolicy(routeHostname string, accessControlPolicy gwpav1alpha1.AccessControlPolicy) *gwpav1alpha1.AccessControlConfig {
	if len(accessControlPolicy.Spec.Hostnames) == 0 {
		return nil
	}

	for i := range accessControlPolicy.Spec.Hostnames {
		hostname := string(accessControlPolicy.Spec.Hostnames[i].Hostname)

		switch {
		case routeHostname == hostname:
			return ComputeAccessControlConfig(accessControlPolicy.Spec.Hostnames[i].Config, accessControlPolicy.Spec.DefaultConfig)

		case strings.HasPrefix(routeHostname, "*"):
			if gwutils.HostnameMatchesWildcardHostname(hostname, routeHostname) {
				return ComputeAccessControlConfig(accessControlPolicy.Spec.Hostnames[i].Config, accessControlPolicy.Spec.DefaultConfig)
			}

		case strings.HasPrefix(hostname, "*"):
			if gwutils.HostnameMatchesWildcardHostname(routeHostname, hostname) {
				return ComputeAccessControlConfig(accessControlPolicy.Spec.Hostnames[i].Config, accessControlPolicy.Spec.DefaultConfig)
			}
		}
	}

	return nil
}

// GetAccessControlConfigIfHTTPRouteMatchesPolicy returns the access control config if the HTTP route matches the policy
func GetAccessControlConfigIfHTTPRouteMatchesPolicy(routeMatch gwv1.HTTPRouteMatch, accessControlPolicy gwpav1alpha1.AccessControlPolicy) *gwpav1alpha1.AccessControlConfig {
	if len(accessControlPolicy.Spec.HTTPAccessControls) == 0 {
		return nil
	}

	for _, hr := range accessControlPolicy.Spec.HTTPAccessControls {
		if reflect.DeepEqual(routeMatch, hr.Match) {
			return ComputeAccessControlConfig(hr.Config, accessControlPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// GetAccessControlConfigIfGRPCRouteMatchesPolicy returns the access control config if the GRPC route matches the policy
func GetAccessControlConfigIfGRPCRouteMatchesPolicy(routeMatch gwv1alpha2.GRPCRouteMatch, accessControlPolicy gwpav1alpha1.AccessControlPolicy) *gwpav1alpha1.AccessControlConfig {
	if len(accessControlPolicy.Spec.GRPCAccessControls) == 0 {
		return nil
	}

	for _, gr := range accessControlPolicy.Spec.GRPCAccessControls {
		if reflect.DeepEqual(routeMatch, gr.Match) {
			return ComputeAccessControlConfig(gr.Config, accessControlPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// ComputeAccessControlConfig computes the access control config based on the config and default config
func ComputeAccessControlConfig(config *gwpav1alpha1.AccessControlConfig, defaultConfig *gwpav1alpha1.AccessControlConfig) *gwpav1alpha1.AccessControlConfig {
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

func mergeConfig(config *gwpav1alpha1.AccessControlConfig, defaultConfig *gwpav1alpha1.AccessControlConfig) *gwpav1alpha1.AccessControlConfig {
	cfgCopy := config.DeepCopy()

	if cfgCopy.EnableXFF == nil {
		if defaultConfig.EnableXFF != nil {
			// use port config
			cfgCopy.EnableXFF = defaultConfig.EnableXFF
		} else {
			// all nil, set to false
			cfgCopy.EnableXFF = pointer.Bool(false)
		}
	}

	if cfgCopy.Message == nil {
		if defaultConfig.Message != nil {
			// use port config
			cfgCopy.Message = defaultConfig.Message
		} else {
			// all nil, set to false
			cfgCopy.Message = pointer.String("")
		}
	}

	if cfgCopy.StatusCode == nil {
		if defaultConfig.StatusCode != nil {
			// use port config
			cfgCopy.StatusCode = defaultConfig.StatusCode
		} else {
			// all nil, set to false
			cfgCopy.StatusCode = pointer.Int32(403)
		}
	}

	return cfgCopy
}

func setDefaultValues(config *gwpav1alpha1.AccessControlConfig) *gwpav1alpha1.AccessControlConfig {
	cfg := config.DeepCopy()

	if cfg.EnableXFF == nil {
		cfg.EnableXFF = pointer.Bool(false)
	}

	if cfg.StatusCode == nil {
		cfg.StatusCode = pointer.Int32(403)
	}

	if cfg.Message == nil {
		cfg.Message = pointer.String("")
	}

	return cfg
}
