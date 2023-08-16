package utils

import "fmt"

// IngressCodebasePath returns the path to the ingress codebase.
func IngressCodebasePath() string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/ingress

	return GetDefaultIngressPath()
}

// NamespacedIngressCodebasePath returns the path to the ingress codebase for the given namespace.
func NamespacedIngressCodebasePath(namespace string) string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/nsig/{{ .Namespace }}

	//return util.EvaluateTemplate(commons.NamespacedIngressPathTemplate, struct {
	//	Region    string
	//	Zone      string
	//	Group     string
	//	Cluster   string
	//	Namespace string
	//}{
	//	Region:    c.Cluster.Region,
	//	Zone:      c.Cluster.Zone,
	//	Group:     c.Cluster.Group,
	//	Cluster:   c.Cluster.Name,
	//	Namespace: namespace,
	//})

	return fmt.Sprintf("/local/nsig/%s", namespace)
}

// GetDefaultServicesPath returns the path to the services codebase.
func GetDefaultServicesPath() string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/services

	//return util.EvaluateTemplate(commons.ServicePathTemplate, struct {
	//	Region  string
	//	Zone    string
	//	Group   string
	//	Cluster string
	//}{
	//	Region:  c.Cluster.Region,
	//	Zone:    c.Cluster.Zone,
	//	Group:   c.Cluster.Group,
	//	Cluster: c.Cluster.Name,
	//})

	return "/local/services"
}

// GetDefaultIngressPath returns the path to the ingress codebase.
func GetDefaultIngressPath() string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/ingress

	//return util.EvaluateTemplate(commons.IngressPathTemplate, struct {
	//	Region  string
	//	Zone    string
	//	Group   string
	//	Cluster string
	//}{
	//	Region:  c.Cluster.Region,
	//	Zone:    c.Cluster.Zone,
	//	Group:   c.Cluster.Group,
	//	Cluster: c.Cluster.Name,
	//})

	return "/local/ingress"
}

// GatewayCodebasePath get the codebase URL for the gateway in specified namespace
// inherit hierarchy: /base/gateways -> /local/gateways -> /local/gw/[ns]
func GatewayCodebasePath(namespace string) string {
	return fmt.Sprintf("/local/gw/%s", namespace)
}

// GetDefaultGatewaysPath returns the path to the gateways codebase.
// inherit hierarchy: /base/gateways -> /local/gateways -> /local/gw/[ns]
func GetDefaultGatewaysPath() string {
	return "/local/gateways"
}
