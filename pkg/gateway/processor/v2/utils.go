package v2

import (
	"k8s.io/utils/ptr"

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
