package ratelimit

import (
	"reflect"
	"strings"

	"k8s.io/utils/pointer"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// GetRateLimitIfRouteHostnameMatchesPolicy returns the rate limit config if the route hostname matches the policy
func GetRateLimitIfRouteHostnameMatchesPolicy(routeHostname string, rateLimitPolicy *gwpav1alpha1.RateLimitPolicy) *gwpav1alpha1.L7RateLimit {
	if len(rateLimitPolicy.Spec.Hostnames) == 0 {
		return nil
	}

	for i := range rateLimitPolicy.Spec.Hostnames {
		hostname := string(rateLimitPolicy.Spec.Hostnames[i].Hostname)

		switch {
		case routeHostname == hostname:
			return ComputeRateLimitConfig(rateLimitPolicy.Spec.Hostnames[i].Config, rateLimitPolicy.Spec.DefaultConfig)

		case strings.HasPrefix(routeHostname, "*"):
			if gwutils.HostnameMatchesWildcardHostname(hostname, routeHostname) {
				return ComputeRateLimitConfig(rateLimitPolicy.Spec.Hostnames[i].Config, rateLimitPolicy.Spec.DefaultConfig)
			}

		case strings.HasPrefix(hostname, "*"):
			if gwutils.HostnameMatchesWildcardHostname(routeHostname, hostname) {
				return ComputeRateLimitConfig(rateLimitPolicy.Spec.Hostnames[i].Config, rateLimitPolicy.Spec.DefaultConfig)
			}
		}
	}

	return nil
}

// GetRateLimitIfHTTPRouteMatchesPolicy returns the rate limit config if the HTTP route matches the policy
func GetRateLimitIfHTTPRouteMatchesPolicy(routeMatch gwv1.HTTPRouteMatch, rateLimitPolicy *gwpav1alpha1.RateLimitPolicy) *gwpav1alpha1.L7RateLimit {
	if len(rateLimitPolicy.Spec.HTTPRateLimits) == 0 {
		return nil
	}

	for _, hr := range rateLimitPolicy.Spec.HTTPRateLimits {
		if reflect.DeepEqual(routeMatch, hr.Match) {
			return ComputeRateLimitConfig(hr.Config, rateLimitPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// GetRateLimitIfGRPCRouteMatchesPolicy returns the rate limit config if the GRPC route matches the policy
func GetRateLimitIfGRPCRouteMatchesPolicy(routeMatch gwv1.GRPCRouteMatch, rateLimitPolicy *gwpav1alpha1.RateLimitPolicy) *gwpav1alpha1.L7RateLimit {
	if len(rateLimitPolicy.Spec.GRPCRateLimits) == 0 {
		return nil
	}

	for _, gr := range rateLimitPolicy.Spec.GRPCRateLimits {
		if reflect.DeepEqual(routeMatch, gr.Match) {
			return ComputeRateLimitConfig(gr.Config, rateLimitPolicy.Spec.DefaultConfig)
		}
	}

	return nil
}

// GetRateLimitIfPortMatchesPolicy returns true if the port matches the rate limit policy
func GetRateLimitIfPortMatchesPolicy(port gwv1.PortNumber, rateLimitPolicy *gwpav1alpha1.RateLimitPolicy) *int64 {
	if len(rateLimitPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range rateLimitPolicy.Spec.Ports {
		if port == policyPort.Port {
			if policyPort.BPS != nil {
				return policyPort.BPS
			}

			return rateLimitPolicy.Spec.DefaultBPS
		}
	}

	return nil
}

// ComputeRateLimitConfig computes the rate limit config based on the config and default config
func ComputeRateLimitConfig(rateLimit *gwpav1alpha1.L7RateLimit, defaultRateLimit *gwpav1alpha1.L7RateLimit) *gwpav1alpha1.L7RateLimit {
	switch {
	case rateLimit == nil && defaultRateLimit == nil:
		return nil
	case rateLimit == nil && defaultRateLimit != nil:
		return setDefaultValues(defaultRateLimit.DeepCopy())
	case rateLimit != nil && defaultRateLimit == nil:
		return setDefaultValues(rateLimit.DeepCopy())
	case rateLimit != nil && defaultRateLimit != nil:
		return mergeConfig(rateLimit, defaultRateLimit)
	}

	return nil
}

func mergeConfig(config *gwpav1alpha1.L7RateLimit, defaultConfig *gwpav1alpha1.L7RateLimit) *gwpav1alpha1.L7RateLimit {
	cfgCopy := config.DeepCopy()

	if cfgCopy.Mode == nil {
		if defaultConfig.Mode != nil {
			cfgCopy.Mode = defaultConfig.Mode
		} else {
			cfgCopy.Mode = rateLimitPolicyModePointer(gwpav1alpha1.RateLimitPolicyModeLocal)
		}
	}

	if cfgCopy.Backlog == nil {
		if defaultConfig.Backlog != nil {
			cfgCopy.Backlog = defaultConfig.Backlog
		} else {
			cfgCopy.Backlog = pointer.Int32(10)
		}
	}

	if cfgCopy.Burst == nil {
		if defaultConfig.Burst != nil {
			cfgCopy.Burst = defaultConfig.Burst
		} else {
			cfgCopy.Burst = &cfgCopy.Requests
		}
	}

	if cfgCopy.ResponseStatusCode == nil {
		if defaultConfig.ResponseStatusCode != nil {
			cfgCopy.ResponseStatusCode = defaultConfig.ResponseStatusCode
		} else {
			cfgCopy.ResponseStatusCode = pointer.Int32(429)
		}
	}

	if len(config.ResponseHeadersToAdd) == 0 && len(defaultConfig.ResponseHeadersToAdd) > 0 {
		cfgCopy.ResponseHeadersToAdd = make([]gwv1.HTTPHeader, 0)
		cfgCopy.ResponseHeadersToAdd = append(cfgCopy.ResponseHeadersToAdd, defaultConfig.ResponseHeadersToAdd...)
	}

	return cfgCopy
}

func setDefaultValues(rateLimit *gwpav1alpha1.L7RateLimit) *gwpav1alpha1.L7RateLimit {
	result := rateLimit.DeepCopy()

	if result.Mode == nil {
		result.Mode = rateLimitPolicyModePointer(gwpav1alpha1.RateLimitPolicyModeLocal)
	}

	if result.Backlog == nil {
		result.Backlog = pointer.Int32(10)
	}

	if result.Burst == nil {
		result.Burst = &result.Requests
	}

	if result.ResponseStatusCode == nil {
		result.ResponseStatusCode = pointer.Int32(429)
	}

	return result
}

func rateLimitPolicyModePointer(mode gwpav1alpha1.RateLimitPolicyMode) *gwpav1alpha1.RateLimitPolicyMode {
	return &mode
}
