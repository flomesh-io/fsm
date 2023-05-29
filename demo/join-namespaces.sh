#!/bin/bash



# This script joins the list of namespaces to the existing service mesh.
# This is a helper script part of brownfield FSM demo.



set -aueo pipefail

# shellcheck disable=SC1091
source .env


./bin/fsm namespace add "${BOOKBUYER_NAMESPACE:-bookbuyer}"         --mesh-name "${MESH_NAME:-fsm}"
./bin/fsm namespace add "${BOOKSTORE_NAMESPACE:-bookstore}"         --mesh-name "${MESH_NAME:-fsm}"
./bin/fsm namespace add "${BOOKTHIEF_NAMESPACE:-bookthief}"         --mesh-name "${MESH_NAME:-fsm}"
./bin/fsm namespace add "${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}" --mesh-name "${MESH_NAME:-fsm}"

kubectl patch meshconfig fsm-mesh-config -n "${K8S_NAMESPACE}" -p '{"spec":{"traffic":{"enablePermissiveTrafficPolicyMode":false}}}'  --type=merge


# Create a top level service
echo -e "Deploy bookstore Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  labels:
    app: bookstore
  name: bookstore
  namespace: bookstore
spec:
  ports:
  - name: bookstore-port
    port: 14001
    protocol: TCP
    targetPort: 14001
  selector:
    app: bookstore
EOF

sleep 3


./demo/rolling-restart.sh
