package v2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/flomesh-io/fsm/pkg/k8s"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func toFGWBackendTargets(endpointSet map[endpointContext]struct{}) []fgwv2.BackendTarget {
	targets := make([]fgwv2.BackendTarget, 0)
	for ep := range endpointSet {
		targets = append(targets, fgwv2.BackendTarget{
			Address: ep.address,
			Port:    ptr.To(ep.port),
			Weight:  1,
		})
	}

	return targets
}

func backendWeight(bk gwv1.BackendRef) int32 {
	if bk.Weight != nil {
		return *bk.Weight
	}

	return 1
}

func isHeadlessServiceWithoutSelector(service *corev1.Service) bool {
	return k8s.IsHeadlessService(service) && len(service.Spec.Selector) == 0
}

func parseFilterConfig(filterConfig string) (vals map[string]interface{}) {
	if filterConfig == "" {
		return map[string]interface{}{}
	}

	err := yaml.Unmarshal([]byte(filterConfig), &vals)
	if err != nil {
		log.Error().Msgf("Failed to parse filter config: %v", err)
	}

	if len(vals) == 0 {
		vals = map[string]interface{}{}
	}

	return vals
}
