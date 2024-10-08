package v2

import (
	"net"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/utils/cidr"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func toFGWBackendTargets(endpointSet map[endpointContext]struct{}) []fgwv2.BackendTarget {
	targets := make([]fgwv2.BackendTarget, 0)

	var isolationCidrs []*cidr.CIDR
	if mc := catalog.GetMeshCatalog(); mc != nil {
		isolationCidrs = mc.GetIsolationCidrs()
	}

	for ep := range endpointSet {
		isolation := false
		if len(isolationCidrs) > 0 {
			for _, isolationCidr := range isolationCidrs {
				if isolationCidr.Has(net.ParseIP(ep.address)) {
					isolation = true
					break
				}
			}
		}
		if !isolation {
			targets = append(targets, fgwv2.BackendTarget{
				Address: ep.address,
				Port:    ptr.To(ep.port),
				Weight:  1,
			})
		}
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
