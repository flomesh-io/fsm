package injector

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
)

const (
	// OutboundPortExclusionListAnnotation is the annotation used for outbound port exclusions
	OutboundPortExclusionListAnnotation = "flomesh.io/outbound-port-exclusion-list"

	// InboundPortExclusionListAnnotation is the annotation used for inbound port exclusions
	InboundPortExclusionListAnnotation = "flomesh.io/inbound-port-exclusion-list"

	// OutboundIPRangeExclusionListAnnotation is the annotation used for outbound IP range exclusions
	OutboundIPRangeExclusionListAnnotation = "flomesh.io/outbound-ip-range-exclusion-list"

	// OutboundIPRangeInclusionListAnnotation is the annotation used for outbound IP range inclusions
	OutboundIPRangeInclusionListAnnotation = "flomesh.io/outbound-ip-range-inclusion-list"
)

// GetPortExclusionListForPod gets a list of ports to exclude from sidecar traffic interception for the given
// pod and annotation kind.
//
// Ports are excluded from sidecar interception when the pod is explicitly annotated with a single or
// comma separate list of ports.
//
// The kind of exclusion (inbound vs outbound) is determined by the specified annotation.
//
// The function returns an error when it is unable to determine whether ports need to be excluded from outbound sidecar interception.
func GetPortExclusionListForPod(pod *corev1.Pod, annotation string) ([]int, error) {
	var ports []int

	portsToExcludeStr, ok := pod.Annotations[annotation]
	if !ok {
		// No port exclusion annotation specified
		return ports, nil
	}

	log.Trace().Msgf("Pod with UID %s has port exclusion annotation: '%s:%s'", pod.UID, annotation, portsToExcludeStr)
	portsToExclude := strings.Split(portsToExcludeStr, ",")
	for _, portStr := range portsToExclude {
		portStr := strings.TrimSpace(portStr)
		portVal, err := strconv.Atoi(portStr)
		if err != nil || portVal <= 0 {
			return nil, fmt.Errorf("Invalid port value '%s' specified for annotation '%s'", portStr, annotation)
		}
		ports = append(ports, portVal)
	}

	return ports, nil
}

// MergePortExclusionLists merges the pod specific and global port exclusion lists
func MergePortExclusionLists(podSpecificPortExclusionList, globalPortExclusionList []int) []int {
	portExclusionListMap := mapset.NewSet()
	var portExclusionListMerged []int

	// iterate over the global outbound ports to be excluded
	for _, port := range globalPortExclusionList {
		if addedToSet := portExclusionListMap.Add(port); addedToSet {
			portExclusionListMerged = append(portExclusionListMerged, port)
		}
	}

	// iterate over the pod specific ports to be excluded
	for _, port := range podSpecificPortExclusionList {
		if addedToSet := portExclusionListMap.Add(port); addedToSet {
			portExclusionListMerged = append(portExclusionListMerged, port)
		}
	}

	return portExclusionListMerged
}

// GetOutboundIPRangeListForPod returns a list of IP ranges to include/exclude from sidecar traffic interception for the given
// pod and annotation kind.
//
// IP ranges are included/excluded from sidecar interception when the pod is explicitly annotated with a single or
// comma separate list of IP CIDR ranges.
//
// The kind of exclusion (inclusion vs exclusion) is determined by the specified annotation.
//
// The function returns an error when it is unable to determine whether IP ranges need to be excluded from outbound sidecar interception.
func GetOutboundIPRangeListForPod(pod *corev1.Pod, annotation string) ([]string, error) {
	ipRangeExclusionsStr, ok := pod.Annotations[annotation]
	if !ok {
		// No port exclusion annotation specified
		return nil, nil
	}

	var ipRanges []string
	log.Trace().Msgf("Pod with UID %s has IP range exclusion annotation: '%s:%s'", pod.UID, annotation, ipRangeExclusionsStr)

	for _, ip := range strings.Split(ipRangeExclusionsStr, ",") {
		ip := strings.TrimSpace(ip)
		if _, _, err := net.ParseCIDR(ip); err != nil {
			return nil, fmt.Errorf("Invalid IP range '%s' specified for annotation '%s'", ip, annotation)
		}
		ipRanges = append(ipRanges, ip)
	}

	return ipRanges, nil
}

// MergeIPRangeLists merges the pod specific and global IP range (exclusion/inclusion) lists
func MergeIPRangeLists(podSpecific, global []string) []string {
	ipSet := mapset.NewSet()
	var ipRanges []string

	for _, ip := range podSpecific {
		if addedToSet := ipSet.Add(ip); addedToSet {
			ipRanges = append(ipRanges, ip)
		}
	}

	for _, ip := range global {
		if addedToSet := ipSet.Add(ip); addedToSet {
			ipRanges = append(ipRanges, ip)
		}
	}

	return ipRanges
}
