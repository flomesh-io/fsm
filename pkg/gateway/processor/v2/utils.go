package v2

import (
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/utils/cidr"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"
)

func toFGWBackendTargets(endpointSet map[endpointContext]struct{}) []fgwv2.BackendTarget {
	targets := make([]fgwv2.BackendTarget, 0)

	var isolationCidrs []*cidr.CIDR
	if mc := catalog.GetMeshCataloger(); mc != nil {
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
				Weight:  ptr.To(int32(1)), // TODO: support setting weight of endpoint in the future
			})
		}
	}

	return targets
}

func isHeadlessServiceWithoutSelector(service *corev1.Service) bool {
	return k8s.IsHeadlessService(service) && len(service.Spec.Selector) == 0
}

func toFGWAppProtocol(appProtocol *string) *string {
	if appProtocol == nil {
		return nil
	}

	for _, prefix := range []string{
		"kubernetes.io/",
		"gateway.networking.k8s.io/",
		"gateway.flomesh.io/",
	} {
		if strings.HasPrefix(*appProtocol, prefix) {
			after, _ := strings.CutPrefix(*appProtocol, prefix)
			return ptr.To(after)
		}
	}

	return appProtocol
}
