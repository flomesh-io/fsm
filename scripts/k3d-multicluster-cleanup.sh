#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

worker_list="cluster-1 cluster-2 cluster-3"
cluster_list="control-plane $worker_list"

# create k3d clusters
for K3D_CLUSTER_NAME in $cluster_list
do
  k3d cluster delete "${K3D_CLUSTER_NAME}"
done