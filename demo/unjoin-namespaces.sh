#!/bin/bash


# This script removes the list of namespaces from the FSM.
# This is a helper script part of the FSM Brownfield Deployment Demo.

set -aueo pipefail

# shellcheck disable=SC1091
source .env


K8S_NAMESPACE="${K8S_NAMESPACE:-fsm-system}"
BOOKBUYER_NAMESPACE="${BOOKBUYER_NAMESPACE:-bookbuyer}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"
BOOKTHIEF_NAMESPACE="${BOOKTHIEF_NAMESPACE:-bookthief}"
BOOKWAREHOUSE_NAMESPACE="${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}"


./bin/fsm namespace remove "${BOOKWAREHOUSE_NAMESPACE:-bookbuyer}" --mesh-name "${MESH_NAME:-fsm}"
./bin/fsm namespace remove "${BOOKBUYER_NAMESPACE:-bookbuyer}"     --mesh-name "${MESH_NAME:-fsm}"
./bin/fsm namespace remove "${BOOKSTORE_NAMESPACE:-bookbuyer}"     --mesh-name "${MESH_NAME:-fsm}"
./bin/fsm namespace remove "${BOOKTHIEF_NAMESPACE:-bookbuyer}"     --mesh-name "${MESH_NAME:-fsm}"

kubectl patch meshconfig fsm-mesh-config -n "${K8S_NAMESPACE}" -p '{"spec":{"traffic":{"enablePermissiveTrafficPolicyMode":true}}}'  --type=merge


# Create a top level service
echo -e "Deploy bookstore Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service

metadata:
  name: bookstore
  namespace: $BOOKSTORE_NAMESPACE

spec:
  ports:
  - port: 14001
    name: bookstore-port

  selector:
    app: bookstore-v1

EOF


./demo/rolling-restart.sh
