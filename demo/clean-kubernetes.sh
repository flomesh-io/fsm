#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

MESH_NAME="${MESH_NAME:-fsm}"
TIMEOUT="${TIMEOUT:-90s}"

bin/fsm uninstall mesh -f --mesh-name "$MESH_NAME" --fsm-namespace "$K8S_NAMESPACE" --delete-namespace -a

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl delete namespace "$ns" --ignore-not-found --wait --timeout="$TIMEOUT" &
done

# Clean up Hashicorp Vault deployment
kubectl delete deployment vault -n "$K8S_NAMESPACE" --ignore-not-found --wait --timeout="$TIMEOUT" &
kubectl delete service vault -n "$K8S_NAMESPACE" --ignore-not-found --wait --timeout="$TIMEOUT" &

wait
