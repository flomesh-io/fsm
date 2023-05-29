package multicluster

import (
	multiclusterv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/service"
)

// GetExportedRule retrieves the export rule for the given MeshService
func (c *Client) GetExportedRule(svc service.MeshService) (*multiclusterv1alpha1.ServiceExportRule, error) {
	exportedServiceIf, exists, err := c.informers.GetByKey(informers.InformerKeyServiceExport, svc.NamespacedKey())
	if !exists || err != nil {
		return nil, err
	}

	exportedService := exportedServiceIf.(*multiclusterv1alpha1.ServiceExport)

	for _, condition := range exportedService.Status.Conditions {
		if condition.Type == `Valid` && condition.Status != `True` {
			return nil, nil
		}
		if condition.Type == `Conflict` && condition.Status != `False` {
			return nil, nil
		}
	}

	if len(exportedService.Spec.Rules) == 0 {
		return nil, nil
	}

	for _, rule := range exportedService.Spec.Rules {
		if uint16(rule.PortNumber) == svc.Port {
			return &rule, nil
		}
	}

	return nil, nil
}
