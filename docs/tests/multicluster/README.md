# MultiCluster

## Prerequisites
- Setup **dnsmasq** to resolve *.localhost domain to 127.0.0.1
  * Install: `brew install dnsmasq`
  * Create config directory: `mkdir -pv $(brew --prefix)/etc/`
  * Setup *.localhost domain: `echo 'address=/.localhost/127.0.0.1' >> $(brew --prefix)/etc/dnsmasq.conf`
  * Change DNS port: `echo 'port=53' >> $(brew --prefix)/etc/dnsmasq.conf`
  * Autostart dnsmasq - now and after reboot: `sudo brew services start dnsmasq`
  * Create resolver directory: `sudo mkdir -v /etc/resolver`
  * Add your nameserver to resolvers: `sudo bash -c 'echo "nameserver 127.0.0.1" > /etc/resolver/localhost'`

- Install **kubecm**
  * MacOS: `brew install kubecm`, for other platforms, please see [kubecm docs](https://kubecm.cloud/en-us/install)
 
- Install **jq**  
  * MacOS: `brew install jq`, for other platforms, please see [jq docs](https://jqlang.github.io/jq/download//)

- Install **k3d**
  * MacOS: `brew install k3d`, for other platforms, please see [k3d docs](https://k3d.io/#installation)

## Setup test environment:
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