#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods -n "$K8S_NAMESPACE" --selector app=fsm-xnetwork --field-selector spec.nodeName=k3d-c1-server-0 --no-headers | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$K8S_NAMESPACE" -c fsm-xnet -f
