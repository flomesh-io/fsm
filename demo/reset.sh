#!/bin/bash



# This script is part of the Bookstore brownfield deployment demo.
# This helper script resets the demo to what's concidered a begining state.



set -aueo pipefail

# shellcheck disable=SC1091
source .env


./demo/deploy-smi-policies.sh     # Add SMI policies
./demo/unjoin-namespaces.sh       # Remove namespaces from FSM.
./demo/reset-counters.sh          # Reset counters


# Restart these pods to reset their data stores.
kubectl rollout restart deployment -n fsm-system jaeger
kubectl rollout restart deployment -n fsm-system fsm-grafana
kubectl rollout restart deployment -n fsm-system fsm-prometheus


# Clean up any tcpdump files.
rm -rf ./*.pcap
