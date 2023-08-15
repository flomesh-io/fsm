#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

K3D_IMAGE="${K3D_IMAGE:-rancher/k3s:v1.21.11-k3s1}"
K3D_HOST_IP="${K3D_HOST_IP:-10.0.1.21}"

SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
FSM_BIN="${SHELL_FOLDER}/../bin/fsm"

# deploy httpbin in cluster-1 and cluster-3
export NAMESPACE=httpbin
for K3D_CLUSTER_NAME in cluster-1 cluster-3
do
  kubecm switch k3d-${K3D_CLUSTER_NAME}
  echo "Deploying httpbin to cluster $K3D_CLUSTER_NAME ..."
  kubectl create namespace ${NAMESPACE}
  "$FSM_BIN" namespace add ${NAMESPACE}
  kubectl apply -n ${NAMESPACE} -f - <<EOF
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
          image: flomesh/pipy:latest
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

  sleep 3
  kubectl wait --for=condition=ready pod -n ${NAMESPACE} --all --timeout=60s
done

# deploy curl in cluster-2
export NAMESPACE=curl
kubecm switch k3d-cluster-2
echo "Deploying curl to cluster cluster-2 ..."
kubectl create namespace ${NAMESPACE}
"$FSM_BIN" namespace add ${NAMESPACE}
kubectl apply -n ${NAMESPACE} -f - <<EOF
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

sleep 3
kubectl wait --for=condition=ready pod -n ${NAMESPACE} --all --timeout=60s
