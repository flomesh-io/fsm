#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

K3D_HOST_IP="${K3D_HOST_IP:-10.0.1.21}"

CTR_REGISTRY="${CTR_REGISTRY:-localhost:5000/flomesh}"
CTR_TAG="${CTR_TAG:-latest}"
SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
FSM_BIN="${SHELL_FOLDER}/../bin/fsm"
FSM_NAMESPACE="${FSM_NAMESPACE:-fsm}"

worker_list="cluster-1 cluster-2 cluster-3"
cluster_list="control-plane $worker_list"

# install fsm
for K3D_CLUSTER_NAME in $cluster_list
do
  kubecm switch "k3d-${K3D_CLUSTER_NAME}"

  echo "Waiting for cluster $K3D_CLUSTER_NAME to be ready ..."
  sleep 1
  kubectl wait --timeout=180s --for=condition=ready pod --all -n kube-system
  echo "Installing fsm to cluster $K3D_CLUSTER_NAME ..."
  sleep 1

  DNS_SVC_IP="$(kubectl get svc -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].spec.clusterIP}')"

  "$FSM_BIN" install --fsm-namespace "$FSM_NAMESPACE" \
  	--set fsm.fsmIngress.enabled=true \
  	--set fsm.image.registry="$CTR_REGISTRY" \
  	--set fsm.image.tag="$CTR_TAG" \
  	--set fsm.controllerLogLevel=debug \
  	--set fsm.fsmIngress.logLevel=debug \
  	--set=osm.localDNSProxy.enable=true \
    --set=osm.localDNSProxy.primaryUpstreamDNSServerIPAddr="${DNS_SVC_IP}"

  echo "Waiting for fsm to be ready in cluster $K3D_CLUSTER_NAME ..."
  sleep 1
  kubectl wait --timeout=180s --for=condition=ready pod --all -n "$FSM_NAMESPACE"
done
