package v1alpha2

type LoadBalancerAlgorithm string

const (
	LoadBalancerAlgorithmRoundRobin LoadBalancerAlgorithm = "RoundRobin"
	LoadBalancerAlgorithmLeastLoad  LoadBalancerAlgorithm = "LeastLoad"
)
