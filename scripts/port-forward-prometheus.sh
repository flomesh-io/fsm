#!/bin/bash
# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app="fsm-prometheus" -n "$K8S_NAMESPACE" --no-headers | grep 'Running' | awk 'NR==1{print $1}')"

kubectl port-forward "$POD" -n "$K8S_NAMESPACE" 7070:7070 --address 0.0.0.0

