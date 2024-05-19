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
CODEGEN_VERSION="v0.30.0" # Must match k8s.io/client-go version defined in go.mod
go get k8s.io/code-generator@${CODEGEN_VERSION}
CODEGEN_PKG="$(echo `go env GOPATH`/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION})"

echo ">>> ROOT_DIR: ${ROOT_DIR}"
echo ">>> using codegen: ${CODEGEN_PKG}"

source "${CODEGEN_PKG}/kube_codegen.sh"

# Generate tagged register code
#
# USAGE: kube::codegen::gen_registers [FLAGS] <input-dir>
#
# <input-dir>
#   The root directory under which to search for Go files which request code to
#   be generated.  This must be a local path, not a Go package.
#
#   See note at the top about package structure below that.
#
# FLAGS:
#
#   --boilerplate <string = path_to_kube_codegen_boilerplate>
#     An optional override for the header file to insert into generated files.
#
#   --extra-peer-dir <string>
#     An optional list (this flag may be specified multiple times) of "extra"
#     directories to consider during conversion generation.
#
function kube::codegen::gen_registers() {
    local in_dir=""
    local boilerplate="${KUBE_CODEGEN_ROOT}/hack/boilerplate.go.txt"
    local v="${KUBE_VERBOSE:-0}"
    local extra_peers=()

    while [ "$#" -gt 0 ]; do
        case "$1" in
            "--boilerplate")
                boilerplate="$2"
                shift 2
                ;;
            "--extra-peer-dir")
                extra_peers+=("$2")
                shift 2
                ;;
            *)
                if [[ "$1" =~ ^-- ]]; then
                    echo "unknown argument: $1" >&2
                    return 1
                fi
                if [ -n "$in_dir" ]; then
                    echo "too many arguments: $1 (already have $in_dir)" >&2
                    return 1
                fi
                in_dir="$1"
                shift
                ;;
        esac
    done

    if [ -z "${in_dir}" ]; then
        echo "input-dir argument is required" >&2
        return 1
    fi

    (
        # To support running this from anywhere, first cd into this directory,
        # and then install with forced module mode on and fully qualified name.
        cd "${KUBE_CODEGEN_ROOT}"
        BINS=(
            register-gen
        )
        # shellcheck disable=2046 # printf word-splitting is intentional
        GO111MODULE=on go install $(printf "k8s.io/code-generator/cmd/%s " "${BINS[@]}")
    )
    # Go installs in $GOBIN if defined, and $GOPATH/bin otherwise
    gobin="${GOBIN:-$(go env GOPATH)/bin}"

    # Register
    #
    local input_pkgs=()
    while read -r dir; do
        pkg="$(cd "${dir}" && GO111MODULE=on go list -find .)"
        input_pkgs+=("${pkg}")
    done < <(
        ( kube::codegen::internal::grep -l --null \
            -e '+groupName=' \
            -r "${in_dir}" \
            --include '*.go' \
            || true \
        ) | while read -r -d $'\0' F; do dirname "${F}"; done \
          | LC_ALL=C sort -u
    )

    if [ "${#input_pkgs[@]}" != 0 ]; then
        echo "Generating register code for ${#input_pkgs[@]} targets"

        kube::codegen::internal::findz \
            "${in_dir}" \
            -type f \
            -name zz_generated.register.go \
            | xargs -0 rm -f

        "${gobin}/register-gen" \
            -v "${v}" \
            --output-file zz_generated.register.go \
            --go-header-file "${boilerplate}" \
            "${input_pkgs[@]}"
    fi
}

function generate_client() {
  CUSTOM_RESOURCE_NAME=$1
  CUSTOM_RESOURCE_VERSIONS=$2

  echo ">>> Generating client for ${CUSTOM_RESOURCE_NAME} with versions: ${CUSTOM_RESOURCE_VERSIONS}"

  # delete the generated code as this is additive, removed objects will not be cleaned
  rm -rf "${ROOT_DIR}/pkg/gen/client/${CUSTOM_RESOURCE_NAME}"

  kube::codegen::gen_helpers \
    --boilerplate "${ROOT_DIR}/codegen/boilerplate.go.txt" \
    "${ROOT_DIR}/pkg/apis/${CUSTOM_RESOURCE_NAME}"

  kube::codegen::gen_registers \
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

echo "##### Generating networking.flomesh.io client ######"
generate_client "namespacedingress" "v1alpha1"

echo "##### Generating gateway.flomesh.io PolicyAttachment client ######"
generate_client "policyattachment" "v1alpha1"