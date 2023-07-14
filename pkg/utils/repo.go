package utils

import "fmt"

func IngressCodebasePath() string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/ingress

	return GetDefaultIngressPath()
}

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

// GetDefaultGatewaysPath
// inherit hierarchy: /base/gateways -> /local/gateways -> /local/gw/[ns]
func GetDefaultGatewaysPath() string {
	return "/local/gateways"
}
