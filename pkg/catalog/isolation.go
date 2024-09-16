package catalog

import (
	"github.com/flomesh-io/fsm/pkg/utils/cidr"
)

// GetIsolationCidrs returns the isolation cidrs
func (mc *MeshCatalog) GetIsolationCidrs() []*cidr.CIDR {
	// List the isolation policies
	isolationPolicies := mc.policyController.ListIsolationPolicies()
	if isolationPolicies == nil {
		return nil
	}

	var isolationCidrs []*cidr.CIDR
	for _, p := range isolationPolicies {
		if len(p.Spec.CIDR) > 0 {
			for _, isolationCidr := range p.Spec.CIDR {
				if parsedCidr, err := cidr.ParseCIDR(isolationCidr); err == nil {
					isolationCidrs = append(isolationCidrs, parsedCidr)
				}
			}
		}
	}

	return isolationCidrs
}
