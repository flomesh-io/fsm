#!/bin/bash

#
# MIT License
#
# Copyright (c) since 2021,  flomesh.io Authors.
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

if [ -n "$DEBUG" ]; then
	set -x
fi

set -o errexit
set -o nounset
set -o pipefail

DIR=$(cd $(dirname "${BASH_SOURCE}")/.. && pwd -P)
echo "Current DIR is ${DIR}"

CLI_PATH=cli/cmd
FSM_CHART_PATH=charts/fsm
NAMESPACED_INGRESS_CHART_PATH=charts/namespaced-ingress
NAMESPACED_INGRESS_CONTROLLER_PATH=controllers/namespacedingress/v1alpha1
GATEWAY_CHART_PATH=charts/gateway
GATEWAY_CONTROLLER_PATH=controllers/gateway/v1beta1


########################################################
# package fsm chart
########################################################
${HELM_BIN} dependency update ${FSM_CHART_PATH}/
${HELM_BIN} lint ${FSM_CHART_PATH}/
${HELM_BIN} package ${FSM_CHART_PATH}/ -d ${CLI_PATH}/ --app-version="${PACKAGED_APP_VERSION}" --version=${HELM_CHART_VERSION}
mv ${CLI_PATH}/fsm-${HELM_CHART_VERSION}.tgz ${CLI_PATH}/chart.tgz

########################################################
# package namespaced-ingress chart
########################################################
${HELM_BIN} dependency update ${NAMESPACED_INGRESS_CHART_PATH}/
#${HELM_BIN} lint ${NAMESPACED_INGRESS_CHART_PATH}/
${HELM_BIN} package ${NAMESPACED_INGRESS_CHART_PATH}/ -d ${NAMESPACED_INGRESS_CONTROLLER_PATH}/ --app-version="${PACKAGED_APP_VERSION}" --version=${HELM_CHART_VERSION}
mv ${NAMESPACED_INGRESS_CONTROLLER_PATH}/namespaced-ingress-${HELM_CHART_VERSION}.tgz ${NAMESPACED_INGRESS_CONTROLLER_PATH}/chart.tgz

########################################################
# package gateway chart
########################################################
${HELM_BIN} dependency update ${GATEWAY_CHART_PATH}/
#${HELM_BIN} lint ${GATEWAY_CHART_PATH}/
${HELM_BIN} package ${GATEWAY_CHART_PATH}/ -d ${GATEWAY_CONTROLLER_PATH}/ --app-version="${PACKAGED_APP_VERSION}" --version=${HELM_CHART_VERSION}
mv ${GATEWAY_CONTROLLER_PATH}/gateway-${HELM_CHART_VERSION}.tgz ${GATEWAY_CONTROLLER_PATH}/chart.tgz