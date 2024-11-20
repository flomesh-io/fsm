package flb

import (
	"fmt"
	"strings"

	"github.com/flomesh-io/fsm/pkg/constants"

	corev1 "k8s.io/api/core/v1"
)

// serviceIPs returns the list of ingress IP addresses from the Service
func serviceIPs(svc *corev1.Service) []string {
	var ips []string

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			ips = append(ips, ingress.IP)
		}
	}

	return ips
}

func serviceKey(setting *Setting, svc *corev1.Service, port corev1.ServicePort) string {
	return fmt.Sprintf("%s/%s/%s:%d#%s", setting.k8sCluster, svc.Namespace, svc.Name, port.Port, strings.ToUpper(string(port.Protocol)))
}

func secretKey(setting *Setting, ns, secret string) string {
	return fmt.Sprintf("%s/%s/%s", setting.k8sCluster, ns, secret)
}

func lbIPs(addresses []string) []string {
	if len(addresses) == 0 {
		return nil
	}

	//ips := make([]string, 0)
	//for _, addr := range addresses {
	//	if strings.Contains(addr, ":") {
	//		host, _, err := net.SplitHostPort(addr)
	//		if err != nil {
	//			return nil
	//		}
	//		ips = append(ips, host)
	//	} else {
	//		ips = append(ips, addr)
	//	}
	//}

	return addresses
}

func getValidAlgo(value string) string {
	switch value {
	case "rr", "lc", "ch":
		return value
	default:
		log.Warn().Msgf("Invalid ALGO value %q, will use 'rr' as default", value)
		return "rr"
	}
}

func isSupportedProtocol(protocol corev1.Protocol) bool {
	switch protocol {
	case corev1.ProtocolTCP, corev1.ProtocolUDP:
		return true
	default:
		return false
	}
}

func getServiceHash(svc *corev1.Service) string {
	if len(svc.Annotations) == 0 {
		return ""
	}
	return svc.Annotations[constants.FLBHashAnnotation]
}

func getSecretHash(secret *corev1.Secret) string {
	if len(secret.Annotations) == 0 {
		return ""
	}
	return secret.Annotations[constants.FLBHashAnnotation]
}
