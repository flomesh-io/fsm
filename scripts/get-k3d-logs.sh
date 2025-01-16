#!/bin/bash

# This script will iterate $K8S_NAMESPACE pods and store current logs for all container pods
# and any logs for any previous run of a pod, under <dirPath>/logs/<podname>/<[running|previous]_contname>.log
# If <dirPath> doesn't exist, it is created.
# Requires jq (https://stedolan.github.io/jq/)

# Use: ./get-fsm-namespace-logs.sh <dirPath>

# This command will fail on CI as .env does not exist. Can be ignored.
# shellcheck disable=SC1091
source .env > /dev/null 2>&1 

clusterName=${1}
echo "Cluster name: $clusterName"
res=$(docker ps --all --format json | jq "select(.Names | startswith(\"k3d-${clusterName}-\"))" | jq -s .)

dirName=${2}
echo "Directory name: $dirName"
mkdir -p "${dirName}"

# Iterate containers of k3d cluster
while read -r item
do
  container=$(echo "$item" | jq -r '.Names')
  logStorePath="$dirName/logs/$container"
  mkdir -p "$logStorePath"

  echo "Checking $container"

  pid=$(echo "$item" | jq -r '.ID')
  state=$(echo "$item" | jq -r '.State')

  # Get logs for container
  docker logs --details --tail all "${pid}" &> "${logStorePath}/${state}_${container}_${pid}.log"
done < <(echo "$res" | jq -c '.[]')

echo "Done"
exit