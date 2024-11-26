#!/usr/bin/env bash

# Script to generate client-go types and code for FSM's CRDs
#
# Copyright 2020 Flomesh Service Mesh Authors.
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
#
# Copyright SMI SDK for Go authors.
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
#
# shellcheck disable=SC2006,SC2046,SC2116,SC2086,SC1091

set -o errexit
set -o nounset
set -o pipefail

ROOT_PACKAGE="github.com/flomesh-io/fsm"
ROOT_DIR="$(git rev-parse --show-toplevel)"

# get code-generator version from go.sum
CODEGEN_VERSION="v0.31.1" # Must match k8s.io/client-go version defined in go.mod
go get k8s.io/code-generator@${CODEGEN_VERSION}
CODEGEN_PKG="$(echo `go env GOPATH`/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION})"

echo ">>> ROOT_DIR: ${ROOT_DIR}"
echo ">>> using codegen: ${CODEGEN_PKG}"

source "${CODEGEN_PKG}/kube_codegen.sh"

function generate_client() {
  CUSTOM_RESOURCE_NAME=$1
  CUSTOM_RESOURCE_VERSIONS=$2

  echo ">>> Generating client for ${CUSTOM_RESOURCE_NAME} with versions: ${CUSTOM_RESOURCE_VERSIONS}"

  # delete the generated code as this is additive, removed objects will not be cleaned
  rm -rf "${ROOT_DIR}/pkg/gen/client/${CUSTOM_RESOURCE_NAME}"

  kube::codegen::gen_helpers \
    --boilerplate "${ROOT_DIR}/codegen/boilerplate.go.txt" \
    "${ROOT_DIR}/pkg/apis/${CUSTOM_RESOURCE_NAME}"

  kube::codegen::gen_register \
      --boilerplate "${ROOT_DIR}/codegen/boilerplate.go.txt" \
      "${ROOT_DIR}/pkg/apis/${CUSTOM_RESOURCE_NAME}"

  kube::codegen::gen_client \
    --with-watch \
    --one-input-api "${CUSTOM_RESOURCE_NAME}" \
    --output-dir "${ROOT_DIR}/pkg/gen/client/${CUSTOM_RESOURCE_NAME}" \
    --output-pkg "${ROOT_PACKAGE}/pkg/gen/client/${CUSTOM_RESOURCE_NAME}" \
    --boilerplate "${ROOT_DIR}/codegen/boilerplate.go.txt" \
    "${ROOT_DIR}/pkg/apis"
}

echo "##### Generating config.flomesh.io client ######"
generate_client "config" "v1alpha1,v1alpha2,v1alpha3"

echo "##### Generating policy.flomesh.io client ######"
generate_client "policy" "v1alpha1"

echo "##### Generating networking.k8s.io client ######"
generate_client "networking" "v1"

echo "##### Generating multicluster.flomesh.io client ######"
generate_client "multicluster" "v1alpha1"

echo "##### Generating flomesh.io plugin client ######"
generate_client "plugin" "v1alpha1"

echo "##### Generating flomesh.io machine client ######"
generate_client "machine" "v1alpha1"

echo "##### Generating flomesh.io connector client ######"
generate_client "connector" "v1alpha1"

echo "##### Generating flomesh.io xnetwork client ######"
generate_client "xnetwork" "v1alpha1"

echo "##### Generating networking.flomesh.io client ######"
generate_client "namespacedingress" "v1alpha1"

echo "##### Generating gateway.flomesh.io PolicyAttachment client ######"
generate_client "policyattachment" "v1alpha1,v1alpha2"

echo "##### Generating extension.gateway.flomesh.io extension client ######"
generate_client "extension" "v1alpha1"