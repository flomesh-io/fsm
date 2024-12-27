package injector

import (
	"fmt"
	"strconv"
	"strings"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/constants"
)

// iptablesOutboundStaticRules is the list of iptables rules related to outbound traffic interception and redirection
var iptablesOutboundStaticRules = []string{
	// Redirects outbound TCP traffic hitting FSM_PROXY_OUT_REDIRECT chain to Sidecar's outbound listener port
	fmt.Sprintf("-A FSM_PROXY_OUT_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.SidecarOutboundListenerPort),

	// Traffic to the Proxy Admin port flows to the Proxy -- not redirected
	fmt.Sprintf("-A FSM_PROXY_OUT_REDIRECT -p tcp --dport %d -j ACCEPT", constants.SidecarAdminPort),

	// For outbound TCP traffic jump from OUTPUT chain to FSM_PROXY_OUTBOUND chain
	"-A OUTPUT -p tcp -j FSM_PROXY_OUTBOUND",

	// Outbound traffic from Sidecar to the local app over the loopback interface should jump to the inbound proxy redirect chain.
	// So when an app directs traffic to itself via the k8s service, traffic flows as follows:
	// app -> local sidecar's outbound listener -> iptables -> local sidecar's inbound listener -> app
	fmt.Sprintf("-A FSM_PROXY_OUTBOUND -o lo ! -d 127.0.0.1/32 -m owner --uid-owner %d -j FSM_PROXY_IN_REDIRECT", constants.SidecarUID),

	// Outbound traffic from the app to itself over the loopback interface is not be redirected via the proxy.
	// E.g. when app sends traffic to itself via the pod IP.
	fmt.Sprintf("-A FSM_PROXY_OUTBOUND -o lo -m owner ! --uid-owner %d -j RETURN", constants.SidecarUID),

	// Don't redirect Sidecar traffic back to itself, return it to the next chain for processing
	fmt.Sprintf("-A FSM_PROXY_OUTBOUND -m owner --uid-owner %d -j RETURN", constants.SidecarUID),

	// Skip localhost traffic, doesn't need to be routed via the proxy
	"-A FSM_PROXY_OUTBOUND -d 127.0.0.1/32 -j RETURN",
}

// iptablesInboundStaticRules is the list of iptables rules related to inbound traffic interception and redirection
var iptablesInboundStaticRules = []string{
	// Redirects inbound TCP traffic hitting the FSM_PROXY_IN_REDIRECT chain to Sidecar's inbound listener port
	fmt.Sprintf("-A FSM_PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port %d", constants.SidecarInboundListenerPort),

	// For inbound traffic jump from PREROUTING chain to FSM_PROXY_INBOUND chain
	"-A PREROUTING -p tcp -j FSM_PROXY_INBOUND",

	// Skip metrics query traffic being directed to Sidecar's inbound prometheus listener port
	fmt.Sprintf("-A FSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.SidecarPrometheusInboundListenerPort),

	// Skip inbound health probes; These ports will be explicitly handled by listeners configured on the
	// Sidecar proxy IF any health probes have been configured in the Pod Spec.
	// TODO(draychev): Do not add these if no health probes have been defined (https://github.com/flomesh-io/fsm/issues/2243)
	fmt.Sprintf("-A FSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.LivenessProbePort),
	fmt.Sprintf("-A FSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.ReadinessProbePort),
	fmt.Sprintf("-A FSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.StartupProbePort),
	// Skip inbound health probes (originally TCPSocket health probes); requests handled by fsm-healthcheck
	fmt.Sprintf("-A FSM_PROXY_INBOUND -p tcp --dport %d -j RETURN", constants.HealthcheckPort),

	// Redirect remaining inbound traffic to Sidecar
	"-A FSM_PROXY_INBOUND -p tcp -j FSM_PROXY_IN_REDIRECT",
}

// GenerateIptablesCommands generates a list of iptables commands to set up sidecar interception and redirection
func GenerateIptablesCommands(proxyMode configv1alpha3.LocalProxyMode, outboundIPRangeExclusionList []string, outboundIPRangeInclusionList []string, outboundPortExclusionList []int, inboundPortExclusionList []int, networkInterfaceExclusionList []string) string {
	var rules strings.Builder

	fmt.Fprintln(&rules, `# FSM sidecar interception rules
*nat
:FSM_PROXY_INBOUND - [0:0]
:FSM_PROXY_IN_REDIRECT - [0:0]
:FSM_PROXY_OUTBOUND - [0:0]
:FSM_PROXY_OUT_REDIRECT - [0:0]`)
	var cmds []string

	// 1. Create inbound rules
	cmds = append(cmds, iptablesInboundStaticRules...)

	// Ignore inbound traffic on specified interfaces
	for _, iface := range networkInterfaceExclusionList {
		// *Note: it is important to use the insert option '-I' instead of the append option '-A' to ensure the
		// exclusion of traffic to the network interface happens before the rule that redirects traffic to the proxy
		cmds = append(cmds, fmt.Sprintf("-I FSM_PROXY_INBOUND -i %s -j RETURN", iface))
	}

	// 2. Create dynamic inbound ports exclusion rules
	if len(inboundPortExclusionList) > 0 {
		var portExclusionListStr []string
		for _, port := range inboundPortExclusionList {
			portExclusionListStr = append(portExclusionListStr, strconv.Itoa(port))
		}
		inboundPortsToExclude := strings.Join(portExclusionListStr, ",")
		rule := fmt.Sprintf("-I FSM_PROXY_INBOUND -p tcp --match multiport --dports %s -j RETURN", inboundPortsToExclude)
		cmds = append(cmds, rule)
	}

	// 3. Create outbound rules
	cmds = append(cmds, iptablesOutboundStaticRules...)

	if proxyMode == configv1alpha3.LocalProxyModePodIP {
		// For sidecar -> local service container proxying, send traffic to pod IP instead of localhost
		// *Note: it is important to use the insert option '-I' instead of the append option '-A' to ensure the
		// DNAT to the pod ip for sidecar -> localhost traffic happens before the rule that redirects traffic to the proxy
		cmds = append(cmds, fmt.Sprintf("-I OUTPUT -p tcp -o lo -d 127.0.0.1/32 -m owner --uid-owner %d -j DNAT --to-destination $POD_IP", constants.SidecarUID))
	}

	// Ignore outbound traffic in specified interfaces
	for _, iface := range networkInterfaceExclusionList {
		cmds = append(cmds, fmt.Sprintf("-A FSM_PROXY_OUTBOUND -o %s -j RETURN", iface))
	}

	//
	// Create outbound exclusion and inclusion rules.
	// *Note: exclusion rules must be applied before inclusions as order matters
	//

	// 4. Create dynamic outbound IP range exclusion rules
	for _, cidr := range outboundIPRangeExclusionList {
		// *Note: it is important to use the insert option '-I' instead of the append option '-A' to ensure the exclusion
		// rules take precedence over the static redirection rules. Iptables rules are evaluated in order.
		rule := fmt.Sprintf("-A FSM_PROXY_OUTBOUND -d %s -j RETURN", cidr)
		cmds = append(cmds, rule)
	}

	// 5. Create dynamic outbound ports exclusion rules
	if len(outboundPortExclusionList) > 0 {
		var portExclusionListStr []string
		for _, port := range outboundPortExclusionList {
			portExclusionListStr = append(portExclusionListStr, strconv.Itoa(port))
		}
		outboundPortsToExclude := strings.Join(portExclusionListStr, ",")
		rule := fmt.Sprintf("-A FSM_PROXY_OUTBOUND -p tcp --match multiport --dports %s -j RETURN", outboundPortsToExclude)
		cmds = append(cmds, rule)
	}

	// 6. Create dynamic outbound IP range inclusion rules
	if len(outboundIPRangeInclusionList) > 0 {
		// Redirect specified IP ranges to the proxy
		for _, cidr := range outboundIPRangeInclusionList {
			rule := fmt.Sprintf("-A FSM_PROXY_OUTBOUND -d %s -j FSM_PROXY_OUT_REDIRECT", cidr)
			cmds = append(cmds, rule)
		}
		// Remaining traffic not belonging to specified inclusion IP ranges are not redirected
		cmds = append(cmds, "-A FSM_PROXY_OUTBOUND -j RETURN")
	} else {
		// Redirect remaining outbound traffic to the proxy
		cmds = append(cmds, "-A FSM_PROXY_OUTBOUND -j FSM_PROXY_OUT_REDIRECT")
	}

	for _, rule := range cmds {
		fmt.Fprintln(&rules, rule)
	}

	fmt.Fprint(&rules, "COMMIT")

	cmd := fmt.Sprintf(`iptables-restore --noflush <<EOF
%s
EOF
`, rules.String())

	return cmd
}
