#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

./demo/deploy-traffic-specs.sh
./demo/deploy-traffic-target.sh
./demo/deploy-traffic-split.sh


echo -e "Enable SMI Spec policies"
kubectl apply -f - <<EOF
apiVersion: v1
kind: MeshConfig

metadata:
  name: fsm-mesh-config
  namespace: $K8S_NAMESPACE
spec:
  traffic:
    enablePermissiveTrafficPolicyMode: false

EOF
