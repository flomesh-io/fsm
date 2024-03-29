#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
VERSION=${1:-v1}
SVC="bookstore-$VERSION"
DEPLOY_ON_OPENSHIFT="${DEPLOY_ON_OPENSHIFT:-false}"
USE_PRIVATE_REGISTRY="${USE_PRIVATE_REGISTRY:-false}"
KUBE_CONTEXT=$(kubectl config current-context)

kubectl delete deployment "$SVC" -n "$BOOKSTORE_NAMESPACE"  --ignore-not-found

echo -e "Deploy root bookstore Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: bookstore
  namespace: $BOOKSTORE_NAMESPACE
  labels:
    app: bookstore
spec:
  ports:
  - port: 14001
    name: bookstore-port
  selector:
    app: bookstore
EOF

echo -e "Deploy $SVC Service Account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: "$SVC"
  namespace: $BOOKSTORE_NAMESPACE
EOF

if [ "$DEPLOY_ON_OPENSHIFT" = true ] ; then
    oc adm policy add-scc-to-user privileged -z "$SVC" -n "$BOOKSTORE_NAMESPACE"
    if [ "$USE_PRIVATE_REGISTRY" = true ]; then
        oc secrets link "$SVC" "$CTR_REGISTRY_CREDS_NAME" --for=pull -n "$BOOKSTORE_NAMESPACE"
    fi
fi

echo -e "Deploy $SVC Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: $SVC
  namespace: $BOOKSTORE_NAMESPACE
  labels:
    app: bookstore
    version: $VERSION
spec:
  ports:
  - port: 14001
    name: bookstore-port
  selector:
    app: bookstore
    version: $VERSION
EOF

echo -e "Deploy $SVC Deployment"
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $SVC
  namespace: $BOOKSTORE_NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: bookstore
      version: $VERSION
  template:
    metadata:
      labels:
        app: bookstore
        version: $VERSION
    spec:
      serviceAccountName: "$SVC"
      containers:
        - image: "${CTR_REGISTRY}/fsm-demo-bookstore:${CTR_TAG}"
          imagePullPolicy: Always
          name: bookstore
          ports:
            - containerPort: 14001
              name: web
          command: ["/bookstore"]
          args: ["--port", "14001"]
          env:
            - name: IDENTITY
              value: ${SVC}.${KUBE_CONTEXT}
            - name: BOOKWAREHOUSE_NAMESPACE
              value: ${BOOKWAREHOUSE_NAMESPACE}

          # FSM's mutating webhook will rewrite this liveness probe to /fsm-liveness-probe and
          # Sidecar will have a dedicated listener on port 15901 for this liveness probe
          livenessProbe:
            httpGet:
              path: /liveness
              port: 14001
            initialDelaySeconds: 3
            periodSeconds: 3

          # FSM's mutating webhook will rewrite this readiness probe to /fsm-readiness-probe and
          # Sidecar will have a dedicated listener on port 15902 for this readiness probe
          readinessProbe:
            failureThreshold: 10
            httpGet:
              path: /readiness
              port: 14001
              scheme: HTTP

          # FSM's mutating webhook will rewrite this startup probe to /fsm-startup-probe and
          # Sidecar will have a dedicated listener on port 15903 for this startup probe
          startupProbe:
            httpGet:
              path: /startup
              port: 14001
            failureThreshold: 30
            periodSeconds: 5

      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF

kubectl get pods      --no-headers -o wide --selector app=bookstore,version="$VERSION" -n "$BOOKSTORE_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookstore,version="$VERSION" -n "$BOOKSTORE_NAMESPACE"
kubectl get service                -o wide                       -n "$BOOKSTORE_NAMESPACE"

for x in $(kubectl get service -n "$BOOKSTORE_NAMESPACE" --selector app=bookstore,version="$VERSION" --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKSTORE_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
