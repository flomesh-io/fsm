#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

K3D_IMAGE="${K3D_IMAGE:-rancher/k3s:v1.21.11-k3s1}"
K3D_HOST_IP="${K3D_HOST_IP:-10.0.1.21}"
K3D_NETWORK="${K3D_NETWORK:-fsm}"
SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)

k3d_prefix='k3d'
reg_name='registry.localhost'
final_reg_name="$k3d_prefix-$reg_name"
reg_port='5000'

# create registry container unless it already num_of_exists
jq_reg_exists=".[] | select(.name == \"$final_reg_name\")"
jq_reg_running=".[] | select(.name == \"$final_reg_name\" and .State.Running == true)"
num_of_exists=$(k3d registry list -o json | jq "$jq_reg_exists" | jq -s 'length')
if [ "${num_of_exists}" == '0' ]; then
  k3d registry create "$reg_name" --port "127.0.0.1:$reg_port"
else
  num_of_running=$(k3d registry list -o json | jq "$jq_reg_running" | jq -s 'length')
  if [ "${num_of_exists}" == '1' ] && [ "${num_of_running}" == '0' ]; then
    k3d registry delete "$final_reg_name"
    k3d registry create "$reg_name" --port "127.0.0.1:$reg_port"
  fi
fi

api_port=7443
port=8080

worker_list="cluster-1 cluster-2 cluster-3"
cluster_list="control-plane $worker_list"

# create k3d clusters
for K3D_CLUSTER_NAME in $cluster_list
do
  # check if cluster already exists
  # shellcheck disable=SC2086
  jq_cluster_exists=".[] | select(.name == \"$K3D_CLUSTER_NAME\")"
  num_of_cluster_exists=$(k3d cluster list --no-headers -o json | jq "$jq_cluster_exists" | jq -s 'length')
  if [ "$num_of_cluster_exists" == "1" ] ; then
    echo "cluster '$K3D_CLUSTER_NAME' already num_of_exists, skipping creation of cluster."
    continue
  fi

  # create cluster
  k3d cluster create "${K3D_CLUSTER_NAME}" \
    --registry-use $final_reg_name:$reg_port \
    --registry-config "$SHELL_FOLDER/k3d-registry.yaml" \
    --image "$K3D_IMAGE" \
    --servers 1 \
    --agents 0 \
    --api-port "${K3D_HOST_IP}:${api_port}" \
    --port "${api_port}:6443@server:0" \
    --port "${port}:80@server:0" \
    --servers-memory 4g \
    --k3s-arg "--disable=traefik@server:0" \
    --network "$K3D_NETWORK" \
    --wait

  ((api_port=api_port+1))
  ((port=port+1))
done