package multicluster

import (
	multiclusterv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/service"
)

func (c *Client) getGlobalTrafficPolicy(svc service.MeshService) *multiclusterv1alpha1.GlobalTrafficPolicy {
	gblTrafficPolicyIf, exists, err := c.informers.GetByKey(informers.InformerKeyGlobalTrafficPolicy, svc.NamespacedKey())
	if !exists || err != nil {
		return nil
	}

	return gblTrafficPolicyIf.(*multiclusterv1alpha1.GlobalTrafficPolicy)
}

func (c *Client) isLocality(svc service.MeshService) bool {
	gblTrafficPolicy := c.getGlobalTrafficPolicy(svc)
	if gblTrafficPolicy != nil {
		return gblTrafficPolicy.Spec.LbType == multiclusterv1alpha1.LocalityLbType
	}
	return true
}

func (c *Client) getServiceTrafficPolicy(svc service.MeshService) (lbType multiclusterv1alpha1.LoadBalancerType, clusterKeys map[string]int) {
	gblTrafficPolicy := c.getGlobalTrafficPolicy(svc)
	if gblTrafficPolicy != nil {
		lbType = gblTrafficPolicy.Spec.LbType
		if len(gblTrafficPolicy.Spec.Targets) > 0 {
			clusterKeys = make(map[string]int)
			for _, lbt := range gblTrafficPolicy.Spec.Targets {
				if lbt.Weight != nil {
					clusterKeys[lbt.ClusterKey] = *lbt.Weight
				} else {
					clusterKeys[lbt.ClusterKey] = 0
				}
			}
		}
		return
	}
	lbType = multiclusterv1alpha1.LocalityLbType
	return
}

// GetLbWeightForService retrieves load balancer type and weight for service
func (c *Client) GetLbWeightForService(svc service.MeshService) (aa, fo, lc bool, weight int, clusterKeys map[string]int) {
	gblTrafficPolicy := c.getGlobalTrafficPolicy(svc)
	if gblTrafficPolicy != nil {
		if gblTrafficPolicy.Spec.LbType == multiclusterv1alpha1.ActiveActiveLbType {
			aa = true
			if len(gblTrafficPolicy.Spec.Targets) == 0 {
				weight = constants.ClusterWeightAcceptAll
			} else {
				clusterKeys = make(map[string]int)
				for _, lbt := range gblTrafficPolicy.Spec.Targets {
					if string(svc.ServiceImportUID) == lbt.ClusterKey {
						if lbt.Weight != nil {
							weight = *lbt.Weight
							clusterKeys[lbt.ClusterKey] = *lbt.Weight
						} else {
							weight = 0
							clusterKeys[lbt.ClusterKey] = 0
						}
						break
					}
				}
			}
			return
		}
		if gblTrafficPolicy.Spec.LbType == multiclusterv1alpha1.FailOverLbType {
			fo = true
			weight = constants.ClusterWeightFailOver
			clusterKeys = make(map[string]int)
			for _, lbt := range gblTrafficPolicy.Spec.Targets {
				clusterKeys[lbt.ClusterKey] = constants.ClusterWeightFailOver
			}
			return
		}
	}
	lc = true
	return
}
