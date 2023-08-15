#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

K3D_IMAGE="${K3D_IMAGE:-rancher/k3s:v1.21.11-k3s1}"
K3D_HOST_IP="${K3D_HOST_IP:-10.0.1.21}"

# export httpbin service in cluster-1 and cluster-3 to ClusterSet
export NAMESPACE_MESH=httpbin
for K3D_CLUSTER_NAME in cluster-1 cluster-3
do
  kubecm switch k3d-${K3D_CLUSTER_NAME}
  echo "Exporting httpbin service in cluster $K3D_CLUSTER_NAME to ClusterSet ..."
  kubectl apply -f - <<EOF
apiVersion: flomesh.io/v1alpha1
kind: ServiceExport
metadata:
  namespace: ${NAMESPACE_MESH}
  name: httpbin
spec:
  serviceAccountName: "*"
  rules:
    - portNumber: 8080
      path: "/${K3D_CLUSTER_NAME}/httpbin-mesh"
      pathType: Prefix
---
apiVersion: flomesh.io/v1alpha1
kind: ServiceExport
metadata:
  namespace: ${NAMESPACE_MESH}
  name: httpbin-${K3D_CLUSTER_NAME}
spec:
  serviceAccountName: "*"
  rules:
    - portNumber: 8080
      path: "/${K3D_CLUSTER_NAME}/httpbin-mesh-${K3D_CLUSTER_NAME}"
      pathType: Prefix
EOF
sleep 1
done

echo "Waiting for 10 seconds for service export to take effect ..."
sleep 10

for CLUSTER_NAME_INDEX in 1 3
do
  k3D_CLUSTER_NAME=cluster-${CLUSTER_NAME_INDEX}
  ((PORT=8080+CLUSTER_NAME_INDEX))
  kubecm switch k3d-${k3D_CLUSTER_NAME}
  echo "Getting service exported in cluster ${k3D_CLUSTER_NAME}"
  echo '-----------------------------------'
  kubectl get serviceexports.flomesh.io -A
  echo '-----------------------------------'
  curl -s "http://${K3D_HOST_IP}:${PORT}/${k3D_CLUSTER_NAME}/httpbin-mesh"
  curl -s "http://${K3D_HOST_IP}:${PORT}/${k3D_CLUSTER_NAME}/httpbin-mesh-${k3D_CLUSTER_NAME}"
  echo '-----------------------------------'
done