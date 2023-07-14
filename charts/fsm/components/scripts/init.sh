#!/bin/sh

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

REPO_API_PATH=$(jq -r .repoApiPath < /mesh/mesh_config.json)
export REPO_API_ADDR="http://${REPO_SERVICE_ADDR}${REPO_API_PATH}"
export BASE_CODEBASE_ADDR="${REPO_API_ADDR}${BASE_CODEBASE_PATH}"

CONFIG_CLUSTER_REGION=$(jq -r .cluster.region < /mesh/mesh_config.json)
CONFIG_CLUSTER_ZONE=$(jq -r .cluster.zone < /mesh/mesh_config.json)
CONFIG_CLUSTER_GROUP=$(jq -r .cluster.group < /mesh/mesh_config.json)
CONFIG_CLUSTER_NAME=$(jq -r .cluster.name < /mesh/mesh_config.json)
export LOCAL_CLUSTER_PATH="/${CONFIG_CLUSTER_REGION}/${CONFIG_CLUSTER_ZONE}/${CONFIG_CLUSTER_GROUP}/${CONFIG_CLUSTER_NAME}"
export LOCAL_CLUSTER_ADDR="${REPO_API_ADDR}${LOCAL_CLUSTER_PATH}"

echo "BASE_CODEBASE_ADDR=${BASE_CODEBASE_ADDR}"
echo "LOCAL_CLUSTER_ADDR=${LOCAL_CLUSTER_ADDR}"

function read_dir(){
  for file in $(ls $1)
  do
    if [ -d "$1/$file" ]; then
      read_dir "$1/$file"
    else
      echo "$1/$file"
      curl -s -X POST "${BASE_CODEBASE_ADDR}/$1/$file" --data-binary "@$1/$file"
    fi
  done
}

##################################################################################
# Init base ingress codease /api/v1/repo/base/ingress
##################################################################################
export INGRESS_REPO_NAME=ingress
export BASE_INGRESS_CODEBASE="${BASE_CODEBASE_ADDR}/${INGRESS_REPO_NAME}"

curl -s -X POST "${BASE_INGRESS_CODEBASE}"
read_dir ${INGRESS_REPO_NAME}
version=$(curl -s "${BASE_INGRESS_CODEBASE}" | jq -r .version) || 1
echo "Current version: $version"
version=$(( version+1 ))
echo "New version: $version"
curl -s -X POST "${BASE_INGRESS_CODEBASE}" --data "{\"version\": $version}"

##################################################################################
# In-cluster ingress repo /default/default/default/local/ingress derives
#   /base/ingress
##################################################################################
#export IN_CLUSTER_INGRESS_CODEBASE="${LOCAL_CLUSTER_ADDR}/${INGRESS_REPO_NAME}"
#curl -X POST "${IN_CLUSTER_INGRESS_CODEBASE}" --data '{"version": 1, "base":"/base/ingress"}'
#version=$(curl -s "${IN_CLUSTER_INGRESS_CODEBASE}" | jq -r .version) || 1
#version=$(( version+1 ))
#curl -X POST "${IN_CLUSTER_INGRESS_CODEBASE}" --data "{\"version\": $version}"


##################################################################################
# Init base services codease /api/v1/repo/base/services
##################################################################################
export SERVICE_REPO_NAME=services
export BASE_SERVICE_CODEBASE="${BASE_CODEBASE_ADDR}/${SERVICE_REPO_NAME}"

curl -s -X POST "${BASE_SERVICE_CODEBASE}"
read_dir ${SERVICE_REPO_NAME}
version=$(curl -s "${BASE_SERVICE_CODEBASE}" | jq -r .version) || 1
echo "Current version: $version"
version=$(( version+1 ))
echo "New version: $version"
curl -s -X POST "${BASE_SERVICE_CODEBASE}" --data "{\"version\": $version}"

##################################################################################
# In-cluster services repo /default/default/default/local/services derives
#   /base/services
##################################################################################
#export IN_CLUSTER_SERVICES_CODEBASE="${LOCAL_CLUSTER_ADDR}/${SERVICE_REPO_NAME}"
#curl -X POST "${IN_CLUSTER_SERVICES_CODEBASE}" --data '{"version": 1, "base":"/base/services"}'
#version=$(curl -s "${IN_CLUSTER_SERVICES_CODEBASE}" | jq -r .version) || 1
#version=$(( version+1 ))
#curl -X POST "${IN_CLUSTER_SERVICES_CODEBASE}" --data "{\"version\": $version}"

echo "DONE!"