package policy

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/utils/pointer"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
)

func newRateLimitConfig(rateLimit *gwpav1alpha1.L7RateLimit) *fgw.RateLimit {
	r := &fgw.RateLimit{
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

func newAccessControlLists(c *gwpav1alpha1.AccessControlConfig) *fgw.AccessControlLists {
	return &fgw.AccessControlLists{
		Blacklist:  c.Blacklist,
		Whitelist:  c.Whitelist,
		EnableXFF:  c.EnableXFF,
		StatusCode: c.StatusCode,
		Message:    c.Message,
	}
}

func newCircuitBreaking(cbCfg *gwpav1alpha1.CircuitBreakingConfig) *fgw.CircuitBreaking {
	return &fgw.CircuitBreaking{
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

func newHealthCheck(hc *gwpav1alpha1.HealthCheckConfig) *fgw.HealthCheck {
	h := &fgw.HealthCheck{
		Interval:    hc.Interval,
		MaxFails:    hc.MaxFails,
		FailTimeout: hc.FailTimeout,
		Path:        hc.Path,
	}

	if len(hc.Matches) > 0 {
		h.Matches = make([]fgw.HealthCheckMatch, 0)
		for _, m := range hc.Matches {
			match := fgw.HealthCheckMatch{
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

func newFaultInjection(fault *gwpav1alpha1.FaultInjectionConfig) *fgw.FaultInjection {
	result := &fgw.FaultInjection{}

	if fault.Delay != nil {
		fd := fault.Delay
		delay := &fgw.FaultInjectionDelay{
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
		result.Abort = &fgw.FaultInjectionAbort{
			Percent: fa.Percent,
			Status:  fa.StatusCode,
			Message: fa.Message,
		}
	}

	return result
}

func newUpstreamCert(cfg *UpstreamTLSConfig) *fgw.UpstreamCert {
	cert := &fgw.UpstreamCert{
		IssuingCA: string(cfg.Secret.Data[corev1.ServiceAccountRootCAKey]),
	}

	certChain := string(cfg.Secret.Data[corev1.TLSCertKey])
	if len(certChain) > 0 {
		cert.CertChain = certChain
	}

	privateKey := string(cfg.Secret.Data[corev1.TLSPrivateKeyKey])
	if len(privateKey) > 0 {
		cert.PrivateKey = privateKey
	}

	return cert
}

func newRetry(cfg *gwpav1alpha1.RetryConfig) *fgw.Retry {
	return &fgw.Retry{
		RetryOn:             strings.Join(cfg.RetryOn, ","),
		NumRetries:          cfg.NumRetries,
		BackoffBaseInterval: cfg.BackoffBaseInterval,
	}
}
