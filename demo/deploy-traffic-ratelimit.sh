#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f - <<EOF
apiVersion: policy.flomesh.io/v1alpha1
kind: UpstreamTrafficSetting
metadata:
  name: http-rate-limit
  namespace: bookstore
spec:
  host: bookstore.bookstore.svc.cluster.local
  rateLimit:
    local:
      tcp:
        connections: 6
        unit: minute
        burst: 6
      http:
        requests: 8
        unit: minute
        burst: 8
  httpRoutes:
    - path: .*
      rateLimit:
        local:
          requests: 9
          unit: minute
          burst: 9
  httpHeaders:
    - headers:
        - name: "header-a"
          value: "header-a-value"
        - name: "header-b"
          value: "header-b-value"
      rateLimit:
        local:
          requests: 3
          unit: minute
          burst: 10
EOF
