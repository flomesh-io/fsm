// Package config defines the constants that are used by multiple other packages within FSM.
package config

const (

	// CNISock defines the sock file
	CNISock = "/var/run/fsm-cni.sock"

	// CNICreatePodURL is the route for cni plugin for creating pod
	CNICreatePodURL = "/v1/cni/create-pod"
	// CNIDeletePodURL is the route for cni plugin for deleting pod
	CNIDeletePodURL = "/v1/cni/delete-pod"

	// FsmPodFibEbpfMap is the mount point of fsm_pod_fib map
	FsmPodFibEbpfMap = "/sys/fs/bpf/fsm_pod_fib"
	// FsmNatFibEbpfMap is the mount point of fsm_nat_fib map
	FsmNatFibEbpfMap = "/sys/fs/bpf/fsm_nat_fib"
)
