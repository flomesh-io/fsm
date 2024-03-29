#!/bin/bash

set -aueo pipefail

rm -rf ./bin/fsm-controller

NAME="fsm-controller"
CGO_ENABLED=0 go build -v -o ./bin/fsm-controller ./cmd/fsm-controller

# GRPC_TRACE=all GRPC_VERBOSITY=DEBUG GODEBUG='http2debug=2,gctrace=1,netdns=go+1'

# We could choose a particular cipher suite like this:
# GRPC_SSL_CIPHER_SUITES=ECDHE-ECDSA-AES256-GCM-SHA384
unset GRPC_SSL_CIPHER_SUITES

# Enable gRPC debug logging
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info

mkdir -p "./certificates/$NAME"
./bin/cert --host="$NAME.$K8S_NAMESPACE.azure.mesh" \
           --caPEMFileIn="./certificates/root-cert.pem" \
           --caKeyPEMFileIn="./certificates/root-key.pem" \
           --keyout "./certificates/$NAME/key.pem" \
           --out "./certificates/$NAME/cert.pem"

./bin/fsm-controller \
    --kubeconfig="$HOME/.kube/config" \
    --certpem="./certificates/ads/cert.pem" \
    --keypem="./certificates/ads/key.pem" \
    --rootcertpem="./certificates/root-cert.pem" \
    --verbosity="info"
