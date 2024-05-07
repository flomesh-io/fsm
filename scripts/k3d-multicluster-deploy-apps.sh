#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
FSM_BIN="${SHELL_FOLDER}/../bin/fsm"

worker_list="cluster-1 cluster-2 cluster-3"

# deploy httpbin in cluster-1 and cluster-3
export HTTPBIN_NAMESPACE=httpbin
for K3D_CLUSTER_NAME in $worker_list
do
  echo "------------------------------------------------------------"
  kubecm switch "k3d-${K3D_CLUSTER_NAME}"
  echo "------------------------------------------------------------"
  echo "Deploying httpbin to cluster $K3D_CLUSTER_NAME ..."
  echo "------------------------------------------------------------"

  kubectl create namespace ${HTTPBIN_NAMESPACE}
  "$FSM_BIN" namespace add ${HTTPBIN_NAMESPACE}
kubectl apply -n ${HTTPBIN_NAMESPACE} -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  labels:
    app: pipy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pipy
  template:
    metadata:
      labels:
        app: pipy
    spec:
      containers:
        - name: pipy
          image: flomesh/pipy:1.1.0-1
          ports:
            - containerPort: 8080
          command:
            - pipy
            - -e
            - |
              pipy()
              .listen(8080)
              .serveHTTP(new Message('Hi, I am from ${K3D_CLUSTER_NAME} and controlled by mesh!\n'))
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
  selector:
    app: pipy
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin-${K3D_CLUSTER_NAME}
spec:
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
  selector:
    app: pipy
EOF
  echo "------------------------------------------------------------"
  sleep 3
  kubectl wait --for=condition=ready pod -n ${HTTPBIN_NAMESPACE} --all --timeout=60s
  echo "------------------------------------------------------------"
done

# deploy curl in cluster-2
export CURL_NAMESPACE=curl
echo "------------------------------------------------------------"
kubecm switch k3d-cluster-2
echo "------------------------------------------------------------"
echo "Deploying curl to cluster cluster-2 ..."
echo "------------------------------------------------------------"
kubectl create namespace ${CURL_NAMESPACE}
"$FSM_BIN" namespace add ${CURL_NAMESPACE}
kubectl apply -n ${CURL_NAMESPACE} -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: curl
---
apiVersion: v1
kind: Service
metadata:
  name: curl
  labels:
    app: curl
    service: curl
spec:
  ports:
    - name: http
      port: 80
  selector:
    app: curl
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: curl
spec:
  replicas: 1
  selector:
    matchLabels:
      app: curl
  template:
    metadata:
      labels:
        app: curl
    spec:
      serviceAccountName: curl
      containers:
      - image: curlimages/curl
        imagePullPolicy: IfNotPresent
        name: curl
        command: ["sleep", "365d"]
EOF
echo "------------------------------------------------------------"
sleep 3
kubectl wait --for=condition=ready pod -n ${CURL_NAMESPACE} --all --timeout=60s
echo "------------------------------------------------------------"