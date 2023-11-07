package loadbalancer

import gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

// GetLoadBalancerTypeIfPortMatchesPolicy returns true if the port matches the load balancer policy
func GetLoadBalancerTypeIfPortMatchesPolicy(port int32, loadBalancerPolicy gwpav1alpha1.LoadBalancerPolicy) *gwpav1alpha1.LoadBalancerType {
	if len(loadBalancerPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, p := range loadBalancerPolicy.Spec.Ports {
		if port == int32(p.Port) {
			return ComputeLoadBalancerType(p.Type, loadBalancerPolicy.Spec.DefaultType)
		}
	}

	return nil
}

// ComputeLoadBalancerType computes the load balancer type based on the port type and default type
func ComputeLoadBalancerType(t *gwpav1alpha1.LoadBalancerType, defaultType *gwpav1alpha1.LoadBalancerType) *gwpav1alpha1.LoadBalancerType {
	switch {
	case t == nil && defaultType == nil:
		return loadBalancerType(gwpav1alpha1.RoundRobinLoadBalancer)
	case t == nil && defaultType != nil:
		return defaultType
	case t != nil:
		return t
	}

	return nil
}

func loadBalancerType(t gwpav1alpha1.LoadBalancerType) *gwpav1alpha1.LoadBalancerType {
	return &t
}
