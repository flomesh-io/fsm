#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

K3D_HOST_IP="${K3D_HOST_IP:-10.0.1.21}"
FSM_NAMESPACE="${FSM_NAMESPACE:-fsm}"

worker_list="cluster-1 cluster-2 cluster-3"

kubecm switch k3d-control-plane
echo "Sleep for a while, waiting for fsm to be ready in cluster control-plane ..."
sleep 10

port=8081

# join clusters
for K3D_CLUSTER_NAME in $worker_list
do
echo "Joining cluster $K3D_CLUSTER_NAME to control-plane cluster ..."
kubectl apply -f - <<EOF
apiVersion: flomesh.io/v1alpha1
kind: Cluster
metadata:
  name: ${K3D_CLUSTER_NAME}
spec:
  gatewayHost: ${K3D_HOST_IP}
  gatewayPort: ${port}
  fsmNamespace: ${FSM_NAMESPACE}
  kubeconfig: |+
$(k3d kubeconfig get "${K3D_CLUSTER_NAME}" | sed 's|^|    |g' | sed "s|0.0.0.0|$K3D_HOST_IP|g")
EOF
((port=port+1))
done
