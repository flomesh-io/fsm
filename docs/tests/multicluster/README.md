# MultiCluster

## Prerequisites
- Setup **dnsmasq** to resolve *.localhost domain to 127.0.0.1
  Please see [dnsmasq docs](../dnsmasq/README.md)

- Install **kubecm**
  * MacOS: `brew install kubecm`, for other platforms, please see [kubecm docs](https://kubecm.cloud/en-us/install)
 
- Install **jq**  
  * MacOS: `brew install jq`, for other platforms, please see [jq docs](https://jqlang.github.io/jq/download//)

- Install **k3d**
  * MacOS: `brew install k3d`, for other platforms, please see [k3d docs](https://k3d.io/#installation)

## Setup test environment:
### Export environment variables:
`export K3D_HOST_IP=[you local ip address]`

### Create clusters
`./scripts/k3d-with-registry-multicluster.sh`

### Build Docker Images and push to local registry:
`make docker-build-fsm`

### Install FSM:
`./scripts/k3d-multicluster-install-fsm.sh`

### Join work nodes to the cluster set:
`./scripts/k3d-multicluster-join-clusters.sh`

### Deploy test applications:
`./scripts/k3d-multicluster-deploy-apps.sh`

### Export services to the cluster set:
`./scripts/k3d-multicluster-export-services.sh`

## Test:
`./scripts/k3d-multicluster-curl-test.sh`