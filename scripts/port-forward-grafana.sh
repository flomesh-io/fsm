#!/bin/bash
# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app="fsm-grafana" -n "$K8S_NAMESPACE" --no-headers | grep 'Running' | awk 'NR==1{print $1}')"

kubectl port-forward "$POD" -n "$K8S_NAMESPACE" 3000:3000 --address 0.0.0.0
