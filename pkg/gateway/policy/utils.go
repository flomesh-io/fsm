package policy

import (
	"fmt"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	"k8s.io/utils/pointer"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func newRateLimitConfig(rateLimit *gwpav1alpha1.L7RateLimit) *routecfg.RateLimit {
	r := &routecfg.RateLimit{
		Mode:               *rateLimit.Mode,
		Backlog:            *rateLimit.Backlog,
		Requests:           rateLimit.Requests,
		Burst:              *rateLimit.Burst,
		StatTimeWindow:     rateLimit.StatTimeWindow,
		ResponseStatusCode: *rateLimit.ResponseStatusCode,
	}

	if len(rateLimit.ResponseHeadersToAdd) > 0 {
		r.ResponseHeadersToAdd = make(map[gwv1beta1.HTTPHeaderName]string)
		for _, header := range rateLimit.ResponseHeadersToAdd {
			r.ResponseHeadersToAdd[header.Name] = header.Value
		}
	}

	return r
}

func newAccessControlLists(c *gwpav1alpha1.AccessControlConfig) *routecfg.AccessControlLists {
	return &routecfg.AccessControlLists{
		Blacklist:  c.Blacklist,
		Whitelist:  c.Whitelist,
		EnableXFF:  c.EnableXFF,
		StatusCode: c.StatusCode,
		Message:    c.Message,
	}
}

func newCircuitBreaking(cbCfg *gwpav1alpha1.CircuitBreakingConfig) *routecfg.CircuitBreaking {
	return &routecfg.CircuitBreaking{
		MinRequestAmount:        cbCfg.MinRequestAmount,
		StatTimeWindow:          cbCfg.StatTimeWindow,
		SlowAmountThreshold:     cbCfg.SlowAmountThreshold,
		SlowRatioThreshold:      cbCfg.SlowRatioThreshold,
		SlowTimeThreshold:       cbCfg.SlowTimeThreshold,
		ErrorAmountThreshold:    cbCfg.ErrorAmountThreshold,
		ErrorRatioThreshold:     cbCfg.ErrorRatioThreshold,
		DegradedTimeWindow:      cbCfg.DegradedTimeWindow,
		DegradedStatusCode:      cbCfg.DegradedStatusCode,
		DegradedResponseContent: cbCfg.DegradedResponseContent,
	}
}

func newHealthCheck(hc *gwpav1alpha1.HealthCheckConfig) *routecfg.HealthCheck {
	h := &routecfg.HealthCheck{
		Interval:    hc.Interval,
		MaxFails:    hc.MaxFails,
		FailTimeout: hc.FailTimeout,
		Path:        hc.Path,
	}

	if len(hc.Matches) > 0 {
		h.Matches = make([]routecfg.HealthCheckMatch, 0)
		for _, m := range hc.Matches {
			match := routecfg.HealthCheckMatch{
				StatusCodes: m.StatusCodes,
				Body:        m.Body,
			}

			if len(m.Headers) > 0 {
				match.Headers = make(map[gwv1beta1.HTTPHeaderName]string)
				for _, header := range m.Headers {
					match.Headers[header.Name] = header.Value
				}
			}

			h.Matches = append(h.Matches, match)
		}
	}

	return h
}

func newFaultInjection(fault *gwpav1alpha1.FaultInjectionConfig) *routecfg.FaultInjection {
	result := &routecfg.FaultInjection{}

	if fault.Delay != nil {
		fd := fault.Delay
		delay := &routecfg.FaultInjectionDelay{
			Percent: fd.Percent,
			Fixed:   fd.Fixed,
			Unit:    fd.Unit,
		}

		if fd.Range != nil {
			delay.Range = pointer.String(fmt.Sprintf("%d-%d", fd.Range.Min, fd.Range.Max))
		}

		result.Delay = delay
	}

	if fault.Abort != nil {
		fa := fault.Abort
		result.Abort = &routecfg.FaultInjectionAbort{
			Percent: fa.Percent,
			Status:  fa.StatusCode,
			Message: fa.Message,
		}
	}

	return result
}
