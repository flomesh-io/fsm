package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/net"
)

// FilterByIPFamily filters the given list of IPs by the IPFamilyPolicy of the given service.
func FilterByIPFamily(ips []string, svc *corev1.Service) ([]string, error) {
	var ipFamilyPolicy corev1.IPFamilyPolicyType
	var ipv4Addresses []string

	for _, ip := range ips {
		if net.IsIPv4String(ip) {
			ipv4Addresses = append(ipv4Addresses, ip)
		}
	}

	if svc.Spec.IPFamilyPolicy != nil {
		ipFamilyPolicy = *svc.Spec.IPFamilyPolicy
	}

	switch ipFamilyPolicy {
	case corev1.IPFamilyPolicySingleStack:
		if svc.Spec.IPFamilies[0] == corev1.IPv4Protocol {
			return ipv4Addresses, nil
		}
	}

	return nil, fmt.Errorf("unhandled ipFamilyPolicy")
}
