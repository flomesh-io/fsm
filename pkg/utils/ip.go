package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/net"
)

// FilterByIPFamily filters the given list of IPs by the IPFamilyPolicy of the given service.
func FilterByIPFamily(ips []string, svc *corev1.Service) ([]string, error) {
	ipFamilyPolicy := corev1.IPFamilyPolicySingleStack
	var ipv4Addresses []string
	var ipv6Addresses []string

	for _, ip := range ips {
		if net.IsIPv4String(ip) {
			ipv4Addresses = append(ipv4Addresses, ip)
		}
		if net.IsIPv6String(ip) {
			ipv6Addresses = append(ipv6Addresses, ip)
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
		if svc.Spec.IPFamilies[0] == corev1.IPv6Protocol {
			return ipv6Addresses, nil
		}
	case corev1.IPFamilyPolicyPreferDualStack:
		if svc.Spec.IPFamilies[0] == corev1.IPv4Protocol {
			ipAddresses := append(ipv4Addresses, ipv6Addresses...)
			return ipAddresses, nil
		}
		if svc.Spec.IPFamilies[0] == corev1.IPv6Protocol {
			ipAddresses := append(ipv6Addresses, ipv4Addresses...)
			return ipAddresses, nil
		}
	case corev1.IPFamilyPolicyRequireDualStack:
		if (len(ipv4Addresses) == 0) || (len(ipv6Addresses) == 0) {
			return nil, fmt.Errorf("one or more IP families did not have addresses available for service with ipFamilyPolicy=RequireDualStack")
		}
		if svc.Spec.IPFamilies[0] == corev1.IPv4Protocol {
			ipAddresses := append(ipv4Addresses, ipv6Addresses...)
			return ipAddresses, nil
		}
		if svc.Spec.IPFamilies[0] == corev1.IPv6Protocol {
			ipAddresses := append(ipv6Addresses, ipv4Addresses...)
			return ipAddresses, nil
		}
	}

	return nil, fmt.Errorf("unhandled ipFamilyPolicy: %s", ipFamilyPolicy)
}
