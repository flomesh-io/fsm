package catalog

import (
	"fmt"
	"strings"

	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

// GetEgressGatewayPolicy returns the Egress gateway policy.
func (mc *MeshCatalog) GetEgressGatewayPolicy() (*trafficpolicy.EgressGatewayPolicy, error) {
	egressGateways := mc.policyController.ListEgressGateways()
	if len(egressGateways) > 0 {
		egressGatewayPolicy := new(trafficpolicy.EgressGatewayPolicy)
		for _, egressGateway := range egressGateways {
			if egressGateway.Spec.GlobalEgressGateways != nil {
				for _, globalGateway := range egressGateway.Spec.GlobalEgressGateways {
					egressGatewayMeshSvc := service.MeshService{
						Name:      globalGateway.Service,
						Namespace: globalGateway.Namespace,
					}
					egressGatewaySvc := mc.kubeController.GetService(egressGatewayMeshSvc)
					if egressGatewaySvc != nil {
						mode, exist := egressGatewaySvc.GetAnnotations()[constants.EgressGatewayModeAnnotation]
						if !exist || (constants.EgressGatewayModeHTTP2Tunnel != mode && constants.EgressGatewayModeSock5 != mode) {
							mode = constants.EgressGatewayModeHTTP2Tunnel
						}
						gatewayConfig := trafficpolicy.EgressGatewayConfig{
							Service:   globalGateway.Service,
							Namespace: globalGateway.Namespace,
							Mode:      mode,
							Weight:    globalGateway.Weight,
							Endpoints: mc.listEndpointsForService(egressGatewayMeshSvc),
						}
						egressGatewayPolicy.Global = append(egressGatewayPolicy.Global, &gatewayConfig)
					}
				}
			}
			if egressGateway.Spec.EgressPolicyGatewayRules != nil {
				for _, rule := range egressGateway.Spec.EgressPolicyGatewayRules {
					egressGatewayRule := new(trafficpolicy.EgressGatewayRule)
					egressGatewayRule.Name = egressGateway.Name
					egressGatewayRule.Namespace = egressGateway.Namespace
					for _, egress := range rule.EgressPolicies {
						egressGatewayRule.EgressPolicies = append(egressGatewayRule.EgressPolicies, trafficpolicy.EgressPolicyConfig{
							Name:      egress.Name,
							Namespace: egress.Namespace,
						})
					}
					for _, gateway := range rule.EgressGateways {
						egressGatewayMeshSvc := service.MeshService{
							Name:      gateway.Service,
							Namespace: gateway.Namespace,
						}
						egressGatewaySvc := mc.kubeController.GetService(egressGatewayMeshSvc)
						if egressGatewaySvc != nil {
							mode, exist := egressGatewaySvc.GetAnnotations()[constants.EgressGatewayModeAnnotation]
							if !exist || (constants.EgressGatewayModeHTTP2Tunnel != mode && constants.EgressGatewayModeSock5 != mode) {
								mode = constants.EgressGatewayModeHTTP2Tunnel
							}
							gatewayConfig := trafficpolicy.EgressGatewayConfig{
								Service:   gateway.Service,
								Namespace: gateway.Namespace,
								Mode:      mode,
								Weight:    gateway.Weight,
								Endpoints: mc.listEndpointsForService(egressGatewayMeshSvc),
							}
							egressGatewayRule.EgressGateways = append(egressGatewayRule.EgressGateways, gatewayConfig)
						}
					}
					egressGatewayPolicy.Rules = append(egressGatewayPolicy.Rules, egressGatewayRule)
				}
			}
		}
		return egressGatewayPolicy, nil
	}
	return nil, nil
}

func (mc *MeshCatalog) getGatewayForEgress(egressPolicy *policyv1alpha1.Egress) *string {
	if egressPolicy == nil {
		return nil
	}

	egressGateways := mc.policyController.ListEgressGateways()
	if len(egressGateways) == 0 {
		return nil
	}

	for index, egressGateway := range egressGateways {
		if egressGateway.Spec.EgressPolicyGatewayRules != nil {
			for _, rule := range egressGateway.Spec.EgressPolicyGatewayRules {
				for _, egress := range rule.EgressPolicies {
					if strings.EqualFold(egress.Namespace, egressPolicy.Namespace) && strings.EqualFold(egress.Name, egressPolicy.Name) {
						ruleName := fmt.Sprintf("%s.%s.%d", egressGateway.Namespace, egressGateway.Name, index)
						return &ruleName
					}
				}
			}
		}
	}

	for _, egressGateway := range egressGateways {
		if egressGateway.Spec.GlobalEgressGateways != nil {
			ruleName := "*"
			return &ruleName
		}
	}

	return nil
}
