package v2

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// ConditionStatus returns the status of the condition for a given node.
func ConditionStatus(n *corev1.Node, ct corev1.NodeConditionType) corev1.ConditionStatus {
	if n == nil {
		return corev1.ConditionUnknown
	}

	for _, c := range n.Status.Conditions {
		if c.Type == ct {
			return c.Status
		}
	}

	return corev1.ConditionUnknown
}

// isNetworkUnavailable returns true if the given node NodeNetworkUnavailable condition status is true.
func isNetworkUnavailable(n *corev1.Node) bool {
	return ConditionStatus(n, corev1.NodeNetworkUnavailable) == corev1.ConditionTrue
}

func availableNetworkNodes(kubeClient kubernetes.Interface) (map[string]bool, bool) {
	existsE4lbNode := false
	nodeCaches := make(map[string]bool)
	if nodes, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, node := range nodes.Items {
			if isNetworkUnavailable(&node) {
				continue
			}
			e4lbEnabled := false
			if len(node.Annotations) > 0 {
				e4lbEnabled = utils.ParseEnabled(node.Annotations[constants.FLBEnabledAnnotation])
			}
			if !existsE4lbNode && e4lbEnabled {
				existsE4lbNode = true
			}
			nodeCaches[node.Name] = e4lbEnabled
		}
	}
	return nodeCaches, existsE4lbNode
}
