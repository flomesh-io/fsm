#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# desired cluster name; default is "fsm"
K3D_CLUSTER_NAME="${K3D_CLUSTER_NAME:-fsm}"
K3D_INGRESS_ENABLE="${K3D_INGRESS_ENABLE:-false}"
K3D_NAMESPACED_INGRESS_ENABLE="${K3D_NAMESPACED_INGRESS_ENABLE:-false}"
K3D_GATEWAY_API_ENABLE="${K3D_GATEWAY_API_ENABLE:-false}"
K3D_FLB_ENABLE="${K3D_FLB_ENABLE:-false}"
K3D_SERVICELB_ENABLE="${K3D_SERVICELB_ENABLE:-false}"
K3D_IMAGE="${K3D_IMAGE:-rancher/k3s:v1.25.16-k3s4}"
CI_INTEGRATION_TEST="${CI_INTEGRATION_TEST:-false}"

# shellcheck disable=SC2086
jq_cluster_exists=".[] | select(.name == \"${K3D_CLUSTER_NAME}\")"
num_of_cluster_exists=$(k3d cluster list --no-headers -o json | jq "${jq_cluster_exists}" | jq -s 'length')
if [ "${num_of_cluster_exists}" = "1" ] ; then
  echo "cluster '${K3D_CLUSTER_NAME}' already num_of_exists, skipping creation of cluster."
  exit 0
fi

k3d_network='fsm'
k3d_prefix='k3d'
reg_name='registry.localhost'
final_reg_name="${k3d_prefix}-${reg_name}"
reg_port='5000'

# create registry container unless it already num_of_exists
jq_reg_exists=".[] | select(.name == \"${final_reg_name}\")"
jq_reg_running=".[] | select(.name == \"${final_reg_name}\" and .State.Running == true)"
num_of_exists=$(k3d registry list -o json | jq "${jq_reg_exists}" | jq -s 'length')
if [ "${num_of_exists}" = '0' ]; then
  # no k3d managed registry found, kill any running registry container and create a new one
  #shellcheck disable=SC2046
  container_id=$(docker ps --format json | jq -r 'select(.Image == "registry:2" and .State == "running") | .ID')
  if [ "${container_id}" != "" ]; then
    docker kill "${container_id}"
  fi
  k3d registry create "${reg_name}" --port "127.0.0.1:${reg_port}"
else
  num_of_running=$(k3d registry list -o json | jq "${jq_reg_running}" | jq -s 'length')
  if [ "${num_of_running}" = '0' ]; then
    # no k3d managed registry found, kill any running registry container and create a new one
    k3d registry delete --all
    #shellcheck disable=SC2046
    container_id=$(docker ps --format json | jq -r 'select(.Image == "registry:2" and .State == "running") | .ID')
    if [ "${container_id}" != "" ]; then
      docker kill "${container_id}"
    fi
    k3d registry create "${reg_name}" --port "127.0.0.1:${reg_port}"
  fi
fi

reg_config="
registries:
  use:
    - ${final_reg_name}:${reg_port}
  config: |
    mirrors:
      'localhost:5000':
        endpoint:
          - http://${final_reg_name}:${reg_port}
"

if [ "${CI_INTEGRATION_TEST}" = "true" ]; then
reg_config="
registries:
  config: |
    mirrors:
      ghcr.io:
      docker.io:
    configs:
      ghcr.io:
        auth:
          username: ${GITHUB_REPO_OWNER}
          password: ${GITHUB_TOKEN}
      docker.io:
        tls:
          insecure_skip_verify: true
        auth:
          username: ${DOCKER_USER}
          password: ${DOCKER_PASS}
"
fi

# create cluster
k3d cluster create --verbose --config - <<EOF
apiVersion: k3d.io/v1alpha5
kind: Simple
metadata:
  name: ${K3D_CLUSTER_NAME}
servers: 1
agents: 0
image: ${K3D_IMAGE}
network: ${k3d_network}
${reg_config}
ports:
  - port: 80:80
    nodeFilters:
      - loadbalancer
  - port: 8090:8090
    nodeFilters:
      - loadbalancer
  - port: 9090:9090
    nodeFilters:
      - loadbalancer
  - port: 7443:7443
    nodeFilters:
      - loadbalancer
  - port: 8443:8443
    nodeFilters:
      - loadbalancer
  - port: 9443:9443
    nodeFilters:
      - loadbalancer
  - port: 3000:3000
    nodeFilters:
      - loadbalancer
  - port: 4000:4000/udp
    nodeFilters:
      - loadbalancer
  - port: 3001:3001
    nodeFilters:
      - loadbalancer
  - port: 4001:4001/udp
    nodeFilters:
      - loadbalancer
  - port: 5053:5053/udp
    nodeFilters:
      - loadbalancer
options:
  k3d:
    wait: true
    timeout: "300s"
    disableLoadbalancer: false
    disableImageVolume: false
    disableRollback: false
    loadbalancer:
      configOverrides:
        - settings.workerConnections=2048
  k3s:
    extraArgs:
      - arg: --disable=traefik
        nodeFilters:
          - server:*
      - arg: --kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%
        nodeFilters:
          - server:*
          - agent:*
      - arg: --kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%
        nodeFilters:
          - server:*
          - agent:*
    nodeLabels:
      - label: ingress-ready=true
        nodeFilters:
          - agent:*
  kubeconfig:
    updateDefaultKubeconfig: true
    switchCurrentContext: true
EOF