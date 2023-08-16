#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

kubecm switch k3d-cluster-2
curl_client="$(kubectl get pod -n curl -l app=curl -o jsonpath='{.items[0].metadata.name}')"

echo "Creating GlobalTrafficPolicy(ActiveActive) for httpbin service ..."
kubectl apply -n httpbin -f  - <<EOF
apiVersion: flomesh.io/v1alpha1
kind: GlobalTrafficPolicy
metadata:
  name: httpbin
spec:
  lbType: ActiveActive
  targets:
    - clusterKey: default/default/default/cluster-1
    - clusterKey: default/default/default/cluster-3
EOF

echo "Testing httpbin service ..."
for i in {1..10};
do
    echo "------------------------------------------------------------"
    echo "$i: Visiting http://httpbin.httpbin:8080/ ..."
    kubectl exec "${curl_client}" -n curl -c curl -- curl -s http://httpbin.httpbin:8080/
    echo "------------------------------------------------------------"
    sleep 1
done
