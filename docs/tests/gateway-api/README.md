# GatewayAPI

## Prerequisites
- Install `jq`  
  On MacOS: `brew install jq`, for other platforms, please see [jq docs](https://jqlang.github.io/jq/download//)


- Setup a Kubernetes cluster, we'll use `k3d`:
  ```shell
  export CTR_REGISTRY=localhost:5000/flomesh
  export CTR_TAG=latest
  ./scripts/k3d-with-registry.sh
  ```

  Please note: it exposes 5 ports
  - `8090`: **HTTP**
  - `9090`: **HTTP**, for cross namespace reference
  - `7443`: **HTTPS**
  - `8443`: **TLS Passthrough**
  - `9443`: **TLS Terminate**
  - `3000`: **TCP**
  - `3001`: **TCP**, for cross namespace reference
  - `4000`: **UDP**
  - `4001`: **UDP**, for cross namespace reference
  
- Make docker images available to the cluster
  ```shell
  make docker-build-fsm
  ```

- Install **fsm**, make sure fsmIngress is **disabled** and fsmGateway is **enabled**

  ```shell
  export CTR_REGISTRY=localhost:5000/flomesh
  export CTR_TAG=latest
  ./bin/fsm install --mesh-name fsm \
        --fsm-namespace fsm \
        --set fsm.fsmIngress.enabled=false \
        --set fsm.fsmGateway.enabled=true \
        --set fsm.image.registry=$CTR_REGISTRY \
        --set fsm.image.tag=$CTR_TAG
  ```

- Install **grcpurl**
  - Binaries

    Download the binary from the [grpcurl releases](https://github.com/fullstorydev/grpcurl/releases) page.

  - Homebrew (macOS)

    On macOS, `grpcurl` is available via Homebrew:
    ```shell
    brew install grpcurl
    ```

  - For more installation methods, please see [grpcurl docs](https://github.com/fullstorydev/grpcurl#installation) for details.


- Setup **dnsmasq** to resolve *.localhost domain to 127.0.0.1
  * Please see [dnsmasq docs](../dnsmasq/README.md)
  
## Test cases

### Deploy Gateway

#### Create namespaces
```shell
kubectl create ns test
kubectl create ns http-route
kubectl create ns http
kubectl create ns grpc-route
kubectl create ns grpc
kubectl create ns tcp-route
kubectl create ns tcp
kubectl create ns udp-route
kubectl create ns udp
kubectl create ns nodeport

kubectl label ns http-route app=http-cross
kubectl label ns grpc-route app=grpc-cross
kubectl label ns tcp-route app=tcp-cross
kubectl label ns udp-route app=udp-cross
```

#### Deploy FSM GatewayClass
It's installed with FSM, no need to install it again.

#### Create certs

- Create CA
```shell
openssl genrsa -out ca.key 2048

openssl req -new -x509 -nodes -days 365000 \
  -key ca.key \
  -out ca.crt \
  -subj '/CN=flomesh.io'
```

- Create Cert for HTTPS
```shell
openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 \
  -keyout https.key -out https.crt \
  -subj "/CN=httptest.localhost" \
  -addext "subjectAltName = DNS:httptest.localhost"
```

- Create Secret for HTTPS Gateway resource
```shell
kubectl -n test create secret tls https-cert --key https.key --cert https.crt
```

- Create ConfigMap for HTTPS Gateway resource(CA certificates)
```shell
kubectl -n test create configmap https-ca --from-file=ca.crt=./ca.crt
```

- Create Cert for gRPC
```shell
openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 \
  -keyout grpc.key -out grpc.crt \
  -subj "/CN=grpctest.localhost" \
  -addext "subjectAltName = DNS:grpctest.localhost"
```

- Create Secret for gRPC Gateway resource
```shell
kubectl -n grpc create secret tls grpc-cert --key grpc.key --cert grpc.crt
```

- Create ConfigMap for customizing the Gateway
```shell
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: test
  name: gateway-config
data:
  values.yaml: |
    fsm:
      gateway:
        replicas: 2
        resources:
          requests:
            cpu: 123m
            memory: 257Mi
          limits:
            cpu: 1314m
            memory: 2048Mi
EOF
```

#### Deploy Gateway
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  namespace: test
  name: test-gw-1
spec:
  gatewayClassName: fsm
  listeners:
    - protocol: HTTP
      port: 80
      name: http
    - protocol: HTTP
      port: 9090
      name: http-cross
      allowedRoutes:
        namespaces:
          from: All
    - protocol: TCP
      port: 3000
      name: tcp
    - protocol: TCP
      port: 3001
      name: tcp-cross
      allowedRoutes:
        namespaces:
          from: Selector
          selector: 
            matchLabels:
              app: tcp-cross
    - protocol: HTTPS
      port: 443
      name: https
      tls:
        certificateRefs:
          - name: https-cert
          - name: grpc-cert
            namespace: grpc
        frontendValidation:
          caCertificateRefs:
            - group: ""
              kind: ConfigMap
              name: https-ca
    - protocol: TLS
      port: 8443
      name: tlsp
      hostname: httptest.localhost
      tls:
        mode: Passthrough
    - protocol: TLS
      port: 9443
      name: tlst
      hostname: httptest.localhost
      tls:
        mode: Terminate
        certificateRefs:
          - name: https-cert
          - name: grpc-cert
            namespace: grpc
        frontendValidation:
          caCertificateRefs:
            - group: ""
              kind: ConfigMap
              name: https-ca
#        options:
#          gateway.flomesh.io/mtls: "true"
    - protocol: UDP
      port: 4000
      name: udp
    - protocol: UDP
      port: 5053
      name: dns
      allowedRoutes:
        namespaces:
          from: All
    - protocol: UDP
      port: 4001
      name: udp-cross
      allowedRoutes:
        namespaces:
          from: Selector
          selector: 
            matchLabels:
              app: udp-cross
  infrastructure:
    annotations:
      xyz: abc
    labels:
      test: demo
    parametersRef:
      group: ""
      kind: ConfigMap
      name: gateway-config
EOF
```

#### Create a ReferenceGrant for the secret
```shell
kubectl -n grpc apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: secret-cross-1
spec:
  from:
    - group: gateway.networking.k8s.io
      kind: Gateway
      namespace: test
  to:
    - group: ""
      kind: Secret
      name: grpc-cert
EOF
```

### Test HTTPRoute - refer to svc in same namespace

#### Deploy a HTTP Service
```shell
kubectl -n test apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  ports:
    - name: pipy
      port: 8080
      targetPort: 8080
      protocol: TCP
  selector:
    app: httpbin
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  labels:
    app: httpbin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin
  template:
    metadata:
      labels:
        app: httpbin
    spec:
      containers:
        - name: pipy
          image: flomesh/pipy:1.5.9
          ports:
            - name: pipy
              containerPort: 8080
          command:
            - pipy
            - -e
            - |
              pipy()
              .listen(8080)
              .serveHTTP(new Message('Hi, I am HTTPRoute!\n'))
          workingDir: /tmp
EOF
```

#### Create a HTTPRoute
```shell
kubectl -n test apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: http-app-1
spec:
  parentRefs:
  - name: test-gw-1
    namespace: test
    port: 80
  hostnames:
  - "httptest.localhost"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /bar
    backendRefs:
    - name: httpbin
      port: 8080
EOF
```

#### Test it:
```shell
❯ curl -iv http://httptest.localhost:8090/bar
*   Trying [::1]:8090...
* Connected to httptest.localhost (::1) port 8090
> GET /bar HTTP/1.1
> Host: httptest.localhost:8090
> User-Agent: curl/8.4.0
> Accept: */*
>
< HTTP/1.1 200 OK
HTTP/1.1 200 OK
< content-length: 20
content-length: 20
< connection: keep-alive
connection: keep-alive

<
Hi, I am HTTPRoute!
* Connection #0 to host httptest.localhost left intact
```

### Test HTTPRoute - refer to svc cross namespace

#### Deploy a HTTP Service
```shell
kubectl -n http apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: httpbin-cross
spec:
  ports:
    - name: pipy
      port: 8080
      targetPort: 8080
      protocol: TCP
  selector:
    app: httpbin-cross
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin-cross
  labels:
    app: httpbin-cross
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin-cross
  template:
    metadata:
      labels:
        app: httpbin-cross
    spec:
      containers:
        - name: pipy
          image: flomesh/pipy:1.5.9
          ports:
            - name: pipy
              containerPort: 8080
          command:
            - pipy
            - -e
            - |
              pipy()
              .listen(8080)
              .serveHTTP(new Message('Hi, I am HTTPRoute!\n'))
          workingDir: /tmp
EOF
```

#### Create a HTTPRoute
```shell
kubectl -n http-route apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: http-cross-1
spec:
  parentRefs:
  - name: test-gw-1
    namespace: test
    port: 9090
  hostnames:
  - "httptest.localhost"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /cross
    backendRefs:
    - name: httpbin-cross
      namespace: http
      port: 8080
EOF
```

#### Create a ReferenceGrant
```shell
kubectl -n http apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: http-cross-1
spec:
  from:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      namespace: http-route
  to:
    - group: ""
      kind: Service
      name: httpbin-cross
EOF
```

#### Test it:
```shell
❯ curl -iv http://httptest.localhost:9090/cross
*   Trying [::1]:9090...
* Connected to httptest.localhost (::1) port 9090
> GET /cross HTTP/1.1
> Host: httptest.localhost:9090
> User-Agent: curl/8.4.0
> Accept: */*
>
< HTTP/1.1 200 OK
HTTP/1.1 200 OK
< content-length: 20
content-length: 20
< connection: keep-alive
connection: keep-alive

<
Hi, I am HTTPRoute!
* Connection #0 to host httptest.localhost left intact
```

### Test GRPCRoute - refer to svc in the same namespace
#### Step 1: Create a Kubernetes `Deployment` for gRPC app

- Deploy the gRPC app

```shell
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: grpcbin
  namespace: test
  name: grpcbin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grpcbin
  template:
    metadata:
      labels:
        app: grpcbin
    spec:
      containers:
        - image: flomesh/grpcbin
          resources:
            limits:
              cpu: 100m
              memory: 100Mi
            requests:
              cpu: 50m
              memory: 50Mi
          name: grpcbin
          ports:
            - name: grpc
              containerPort: 9000
EOF
```    

#### Step 2: Create the Kubernetes `Service` for the gRPC app

- You can use the following example manifest to create a service of type ClusterIP.

```shell
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  labels:
    app: grpcbin
  namespace: test
  name: grpcbin
spec:
  ports:
  - name: grpc
    port: 9000
    protocol: TCP
    targetPort: 9000
  selector:
    app: grpcbin
  type: ClusterIP
EOF
```

#### Create a GRPCRoute
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: GRPCRoute
metadata:
  name: grpc-app-1
  namespace: test
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 80  
  hostnames:
    - grpctest.localhost
  rules:
  - matches:
    - method:
        service: hello.HelloService
        method: SayHello
    backendRefs:
    - name: grpcbin
      port: 9000
EOF
```

#### Test it:
```shell
grpcurl -vv -plaintext -d '{"greeting":"Flomesh"}' grpctest.localhost:8090 hello.HelloService/SayHello

Resolved method descriptor:
rpc SayHello ( .hello.HelloRequest ) returns ( .hello.HelloResponse );

Request metadata to send:
host: grpctest.localhost

Response headers received:
content-type: application/grpc

Estimated response size: 15 bytes

Response contents:
{
  "reply": "hello Flomesh"
}

Response trailers received:
(empty)
Sent 1 request and received 1 response
```

### Test GRPCRoute - refer to svc cross namespace
#### Step 1: Create a Kubernetes `Deployment` for gRPC app

- Deploy the gRPC app

```shell
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: grpcbin-cross
  namespace: grpc
  name: grpcbin-cross
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grpcbin-cross
  template:
    metadata:
      labels:
        app: grpcbin-cross
    spec:
      containers:
        - image: flomesh/grpcbin
          resources:
            limits:
              cpu: 100m
              memory: 100Mi
            requests:
              cpu: 50m
              memory: 50Mi
          name: grpcbin
          ports:
            - name: grpc
              containerPort: 9000
EOF
```    

#### Step 2: Create the Kubernetes `Service` for the gRPC app

- You can use the following example manifest to create a service of type ClusterIP.

```shell
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  labels:
    app: grpcbin
  namespace: grpc
  name: grpcbin-cross
spec:
  ports:
  - name: grpc
    port: 9000
    protocol: TCP
    targetPort: 9000
  selector:
    app: grpcbin-cross
  type: ClusterIP
EOF
```

#### Create a GRPCRoute
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: GRPCRoute
metadata:
  name: grpc-cross-1
  namespace: grpc-route
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 9090  
  hostnames:
    - grpctest.localhost
  rules:
  - matches:
    - method:
        service: hello.HelloService
        method: SayHello
    backendRefs:
    - name: grpcbin-cross
      namespace: grpc
      port: 9000
EOF
```

#### Create a ReferenceGrant
```shell
kubectl -n grpc apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: grpc-cross-1
spec:
  from:
    - group: gateway.networking.k8s.io
      kind: GRPCRoute
      namespace: grpc-route
  to:
    - group: ""
      kind: Service
      name: grpcbin-cross
EOF
```

#### Test it:
```shell
❯ grpcurl -vv -plaintext -d '{"greeting":"Flomesh"}' grpctest.localhost:9090 hello.HelloService/SayHello

Resolved method descriptor:
rpc SayHello ( .hello.HelloRequest ) returns ( .hello.HelloResponse );

Request metadata to send:
(empty)

Response headers received:
content-type: application/grpc

Estimated response size: 15 bytes

Response contents:
{
  "reply": "hello Flomesh"
}

Response trailers received:
(empty)
Sent 1 request and received 1 response
```

### Test TCPRoute - refer to svc in the same namespace

#### Deploy the TCPRoute app
```shell
kubectl -n test apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: tcproute
spec:
  ports:
    - name: tcp
      port: 8078
      targetPort: 8078
      protocol: TCP
  selector:
    app: tcproute
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tcproute
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tcproute
  template:
    metadata:
      labels:
        app: tcproute
    spec:
      containers:
        - name: tcp
          image: fortio/fortio:latest
          ports:
            - name: tcp
              containerPort: 8078
          command:
            - fortio
            - tcp-echo
            - -loglevel
            - debug
EOF
```

#### Create TCPRoute
```shell
kubectl -n test apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: tcp-app-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 3000
  rules:
  - backendRefs:
    - name: tcproute
      port: 8078
EOF
```

#### Test it:
```shell
❯ echo "Text to send to TCP" | nc localhost 3000
Text to send to TCP
```

### Test TCPRoute - refer to svc cross namespace

#### Deploy the TCPRoute app
```shell
kubectl -n tcp apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: tcproute-cross
spec:
  ports:
    - name: tcp
      port: 8078
      targetPort: 8078
      protocol: TCP
  selector:
    app: tcproute-cross
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tcproute-cross
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tcproute-cross
  template:
    metadata:
      labels:
        app: tcproute-cross
    spec:
      containers:
        - name: tcp
          image: fortio/fortio:latest
          ports:
            - name: tcp
              containerPort: 8078
          command:
            - fortio
            - tcp-echo
            - -loglevel
            - debug
EOF
```

#### Create TCPRoute
```shell
kubectl -n tcp-route apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: tcp-cross-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 3001
  rules:
  - backendRefs:
    - name: tcproute-cross
      namespace: tcp
      port: 8078
EOF
```

#### Create a ReferenceGrant
```shell
kubectl -n tcp apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: tcp-cross-1
spec:
  from:
    - group: gateway.networking.k8s.io
      kind: TCPRoute
      namespace: tcp-route
  to:
    - group: ""
      kind: Service
      name: tcproute-cross
EOF
```

#### Test it:
```shell
❯ echo "Text to send to TCP" | nc localhost 3001
Text to send to TCP
```

### Test UDPRoute - refer to svc in the same namespace

#### Deploy the UDPRoute app
```shell
kubectl -n test apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: udproute
spec:
  ports:
    - name: udp
      port: 8078
      targetPort: 8078
      protocol: UDP
  selector:
    app: udproute
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: udproute
spec:
  replicas: 1
  selector:
    matchLabels:
      app: udproute
  template:
    metadata:
      labels:
        app: udproute
    spec:
      containers:
        - name: udp
          image: fortio/fortio:latest
          ports:
            - name: udp
              containerPort: 8078
          command:
            - fortio
            - udp-echo
            - -loglevel
            - debug
EOF
```

#### Create UDPRoute
```shell
kubectl -n test apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: UDPRoute
metadata:
  name: udp-app-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 4000
  rules:
  - backendRefs:
    - name: udproute
      port: 8078
EOF
```

#### Test it:
```shell
echo -n "Text to send to UDP" | nc -4u -w1 localhost 4000
```

### Test UDPRoute - refer to svc cross namespace

#### Deploy the UDPRoute app
```shell
kubectl -n udp apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: udproute-cross
spec:
  ports:
    - name: udp
      port: 8078
      targetPort: 8078
      protocol: UDP
  selector:
    app: udproute-cross
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: udproute-cross
spec:
  replicas: 1
  selector:
    matchLabels:
      app: udproute-cross
  template:
    metadata:
      labels:
        app: udproute-cross
    spec:
      containers:
        - name: udp
          image: fortio/fortio:latest
          ports:
            - name: udp
              containerPort: 8078
          command:
            - fortio
            - udp-echo
            - -loglevel
            - debug
EOF
```

#### Create UDPRoute
```shell
kubectl -n udp-route apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: UDPRoute
metadata:
  name: udp-cross-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 4001
  rules:
  - backendRefs:
    - name: udproute-cross
      namespace: udp
      port: 8078
EOF
```

#### Create a ReferenceGrant
```shell
kubectl -n udp apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: udp-cross-1
spec:
  from:
    - group: gateway.networking.k8s.io
      kind: UDPRoute
      namespace: udp-route
  to:
    - group: ""
      kind: Service
      name: udproute-cross
EOF
```

#### Test it:
```shell
echo -n "Text to send to UDP" | nc -4u -w1 localhost 4001
```

### Test UDPRoute - resolve in-cluster DNS

#### Deploy the UDPRoute 
```shell
kubectl -n kube-system apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: UDPRoute
metadata:
  name: udp-dns-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 5053
  rules:
  - name: dns
    backendRefs:
    - name: kube-dns
      port: 53
EOF
```

#### Test it:
```shell
❯ dig @127.0.0.1 -p 5053 kubernetes.default.svc.cluster.local +short
10.43.0.1
```

### Test UDPRoute - resolve DNS with Filter

#### Create a DNSModifier filter config
```shell
kubectl -n kube-system apply -f - <<EOF
apiVersion: extension.gateway.flomesh.io/v1alpha1
kind: DNSModifier
metadata:
  name: dns-mod-1
spec:
  domains:
    - name: google.com
      answer:
        rdata: 11.11.11.11
EOF
```

#### Create a DNSModifier Filter
```shell
kubectl -n kube-system apply -f - <<EOF
apiVersion: extension.gateway.flomesh.io/v1alpha1
kind: Filter
metadata:
  name: test-dns-1
spec:
  type: DNSModifier
  configRef:
    group: extension.gateway.flomesh.io
    kind: DNSModifier
    name: dns-mod-1
EOF
```

#### Create a RouteRuleFilterPolicy to attach the Filter to the UDPRoute
**Note**: The rule name field of UDPRoute must be set, and targetRef must be set to match the rule.
```shell
kubectl -n kube-system apply -f - <<EOF
apiVersion: gateway.flomesh.io/v1alpha2
kind: RouteRuleFilterPolicy
metadata:
  name: dns-policy-1
spec:
  targetRefs:
    - group: gateway.networking.k8s.io
      kind: UDPRoute
      name: udp-dns-1
      rule: dns
  filterRefs:
    - group: extension.gateway.flomesh.io
      kind: Filter
      name: test-dns-1
EOF
```

#### Test it:
```shell
❯ dig @127.0.0.1 -p 5053 google.com +short
11.11.11.11
```

### Test HTTPS - HTTPRoute
#### Create a HTTPRoute and attach to HTTPS port
```shell
kubectl -n test apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: https-app-1
spec:
  parentRefs:
  - name: test-gw-1
    namespace: test
    port: 443
  hostnames:
  - "httptest.localhost"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /bar
    backendRefs:
    - name: httpbin
      port: 8080
EOF
```

#### Test it:
```shell
❯ curl -iv --cacert https.crt https://httptest.localhost:7443/bar
*   Trying 127.0.0.1:7443...
* Connected to httptest.localhost (127.0.0.1) port 7443 (#0)
* ALPN: offers h2,http/1.1
* (304) (OUT), TLS handshake, Client hello (1):
*  CAfile: https.crt
*  CApath: none
* (304) (IN), TLS handshake, Server hello (2):
* (304) (IN), TLS handshake, Unknown (8):
* (304) (IN), TLS handshake, Certificate (11):
* (304) (IN), TLS handshake, CERT verify (15):
* (304) (IN), TLS handshake, Finished (20):
* (304) (OUT), TLS handshake, Finished (20):
* SSL connection using TLSv1.3 / AEAD-AES256-GCM-SHA384
* ALPN: server accepted h2
* Server certificate:
*  subject: CN=httptest.localhost
*  start date: Jul  6 03:41:13 2023 GMT
*  expire date: Jul  5 03:41:13 2024 GMT
*  subjectAltName: host "httptest.localhost" matched cert's "httptest.localhost"
*  issuer: CN=httptest.localhost
*  SSL certificate verify ok.
* using HTTP/2
* h2h3 [:method: GET]
* h2h3 [:path: /bar]
* h2h3 [:scheme: https]
* h2h3 [:authority: httptest.localhost]
* h2h3 [user-agent: curl/7.88.1]
* h2h3 [accept: */*]
* Using Stream ID: 1 (easy handle 0x7fe4c4812e00)
> GET /bar HTTP/2
> Host: httptest.localhost
> user-agent: curl/7.88.1
> accept: */*
>
< HTTP/2 200
HTTP/2 200
< content-length: 20
content-length: 20

<
Hi, I am HTTPRoute!
* Connection #0 to host httptest.localhost left intact
```

### Test HTTPS - GRPCRoute
#### Create a GRPCRoute and attach to HTTPS port
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: GRPCRoute
metadata:
  name: grpcs-app-1
  namespace: test
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 443  
  hostnames:
    - grpctest.localhost
  rules:
  - matches:
    - method:
        service: hello.HelloService
        method: SayHello
    backendRefs:
    - name: grpcbin
      port: 9000
EOF
```

#### Test it:
```shell
❯ grpcurl -vv -cacert grpc.crt -d '{"greeting":"Flomesh"}' grpctest.localhost:7443 hello.HelloService/SayHello

Resolved method descriptor:
rpc SayHello ( .hello.HelloRequest ) returns ( .hello.HelloResponse );

Request metadata to send:
(empty)

Response headers received:
content-type: application/grpc

Estimated response size: 15 bytes

Response contents:
{
  "reply": "hello Flomesh"
}

Response trailers received:
(empty)
Sent 1 request and received 1 response
```

### Test TLS Terminate

#### Create TCPRoute and attach to TLS port
```shell
kubectl -n test apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: tlst-app-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 9443
  rules:
  - backendRefs:
    - name: httpbin
      port: 8080
EOF
```

#### Test it:
```shell
❯ curl -iv --cacert https.crt https://httptest.localhost:9443
* Host httptest.localhost:9443 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:9443...
* Connected to httptest.localhost (::1) port 9443
* ALPN: curl offers h2,http/1.1
* (304) (OUT), TLS handshake, Client hello (1):
*  CAfile: https.crt
*  CApath: none
* (304) (IN), TLS handshake, Server hello (2):
* (304) (IN), TLS handshake, Unknown (8):
* (304) (IN), TLS handshake, Certificate (11):
* (304) (IN), TLS handshake, CERT verify (15):
* (304) (IN), TLS handshake, Finished (20):
* (304) (OUT), TLS handshake, Finished (20):
* SSL connection using TLSv1.3 / AEAD-CHACHA20-POLY1305-SHA256 / [blank] / UNDEF
* ALPN: server did not agree on a protocol. Uses default.
* Server certificate:
*  subject: CN=httptest.localhost
*  start date: Jun 15 10:38:16 2024 GMT
*  expire date: Jun 15 10:38:16 2025 GMT
*  subjectAltName: host "httptest.localhost" matched cert's "httptest.localhost"
*  issuer: CN=httptest.localhost
*  SSL certificate verify ok.
* using HTTP/1.x
> GET / HTTP/1.1
> Host: httptest.localhost:9443
> User-Agent: curl/8.6.0
> Accept: */*
>
< HTTP/1.1 200 OK
HTTP/1.1 200 OK
< content-length: 20
content-length: 20
< connection: keep-alive
connection: keep-alive

<
Hi, I am HTTPRoute!
* Connection #0 to host httptest.localhost left intact
```

### Test TLS Passthrough

#### Create TLSRoute
```shell
kubectl -n test apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TLSRoute
metadata:
  name: tlsp-app-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: test
      port: 8443
  rules:
  - backendRefs:
    - name: bing.com
      port: 443
EOF
```

#### Test it:
```shell
❯ curl https://bing.com -iv --connect-to httptest.localhost:8443:bing:443
*   Trying 204.79.197.200:443...
* Connected to bing.com (204.79.197.200) port 443 (#0)
* ALPN: offers h2,http/1.1
* (304) (OUT), TLS handshake, Client hello (1):
*  CAfile: /etc/ssl/cert.pem
*  CApath: none
* (304) (IN), TLS handshake, Server hello (2):
* TLSv1.2 (IN), TLS handshake, Certificate (11):
* TLSv1.2 (IN), TLS handshake, Server key exchange (12):
* TLSv1.2 (IN), TLS handshake, Server finished (14):
* TLSv1.2 (OUT), TLS handshake, Client key exchange (16):
* TLSv1.2 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.2 (OUT), TLS handshake, Finished (20):
* TLSv1.2 (IN), TLS change cipher, Change cipher spec (1):
* TLSv1.2 (IN), TLS handshake, Finished (20):
* SSL connection using TLSv1.2 / ECDHE-RSA-AES256-GCM-SHA384
* ALPN: server accepted h2
* Server certificate:
*  subject: CN=www.bing.com
*  start date: Feb 16 03:47:45 2023 GMT
*  expire date: Aug 16 03:47:45 2023 GMT
*  subjectAltName: host "bing.com" matched cert's "bing.com"
*  issuer: C=US; O=Microsoft Corporation; CN=Microsoft RSA TLS CA 02
*  SSL certificate verify ok.
* using HTTP/2
* h2h3 [:method: GET]
* h2h3 [:path: /]
* h2h3 [:scheme: https]
* h2h3 [:authority: bing.com]
* h2h3 [user-agent: curl/7.88.1]
* h2h3 [accept: */*]
* Using Stream ID: 1 (easy handle 0x7fabcd010a00)
> GET / HTTP/2
> Host: bing.com
> user-agent: curl/7.88.1
> accept: */*
>
< HTTP/2 301
HTTP/2 301
< cache-control: private
cache-control: private
< content-length: 193
content-length: 193
< content-type: text/html; charset=utf-8
content-type: text/html; charset=utf-8
< location: https://www.bing.com:443/?toWww=1&redig=CA7F9D7F0A6C47FFAFA16A6589BCBB8F
location: https://www.bing.com:443/?toWww=1&redig=CA7F9D7F0A6C47FFAFA16A6589BCBB8F
< set-cookie: SUID=M; domain=bing.com; expires=Thu, 06-Jul-2023 16:44:06 GMT; path=/; HttpOnly
set-cookie: SUID=M; domain=bing.com; expires=Thu, 06-Jul-2023 16:44:06 GMT; path=/; HttpOnly
< set-cookie: MUID=1206AA690AE767CB282BB92F0B3E6622; domain=bing.com; expires=Tue, 30-Jul-2024 04:44:06 GMT; path=/; secure; SameSite=None
set-cookie: MUID=1206AA690AE767CB282BB92F0B3E6622; domain=bing.com; expires=Tue, 30-Jul-2024 04:44:06 GMT; path=/; secure; SameSite=None
< set-cookie: MUIDB=1206AA690AE767CB282BB92F0B3E6622; expires=Tue, 30-Jul-2024 04:44:06 GMT; path=/; HttpOnly
set-cookie: MUIDB=1206AA690AE767CB282BB92F0B3E6622; expires=Tue, 30-Jul-2024 04:44:06 GMT; path=/; HttpOnly
< set-cookie: _EDGE_S=F=1&SID=30355FB840436B0839794CFE419A6AF3; domain=bing.com; path=/; HttpOnly
set-cookie: _EDGE_S=F=1&SID=30355FB840436B0839794CFE419A6AF3; domain=bing.com; path=/; HttpOnly
< set-cookie: _EDGE_V=1; domain=bing.com; expires=Tue, 30-Jul-2024 04:44:06 GMT; path=/; HttpOnly
set-cookie: _EDGE_V=1; domain=bing.com; expires=Tue, 30-Jul-2024 04:44:06 GMT; path=/; HttpOnly
< x-eventid: 64a64696e0854a07b3bd9923f17c56ba
x-eventid: 64a64696e0854a07b3bd9923f17c56ba
< useragentreductionoptout: A7kgTC5xdZ2WIVGZEfb1hUoNuvjzOZX3VIV/BA6C18kQOOF50Q0D3oWoAm49k3BQImkujKILc7JmPysWk3CSjwUAAACMeyJvcmlnaW4iOiJodHRwczovL3d3dy5iaW5nLmNvbTo0NDMiLCJmZWF0dXJlIjoiU2VuZEZ1bGxVc2VyQWdlbnRBZnRlclJlZHVjdGlvbiIsImV4cGlyeSI6MTY4NDg4NjM5OSwiaXNTdWJkb21haW4iOnRydWUsImlzVGhpcmRQYXJ0eSI6dHJ1ZX0=
useragentreductionoptout: A7kgTC5xdZ2WIVGZEfb1hUoNuvjzOZX3VIV/BA6C18kQOOF50Q0D3oWoAm49k3BQImkujKILc7JmPysWk3CSjwUAAACMeyJvcmlnaW4iOiJodHRwczovL3d3dy5iaW5nLmNvbTo0NDMiLCJmZWF0dXJlIjoiU2VuZEZ1bGxVc2VyQWdlbnRBZnRlclJlZHVjdGlvbiIsImV4cGlyeSI6MTY4NDg4NjM5OSwiaXNTdWJkb21haW4iOnRydWUsImlzVGhpcmRQYXJ0eSI6dHJ1ZX0=
< strict-transport-security: max-age=31536000; includeSubDomains; preload
strict-transport-security: max-age=31536000; includeSubDomains; preload
< x-cache: CONFIG_NOCACHE
x-cache: CONFIG_NOCACHE
< accept-ch: Sec-CH-UA-Arch, Sec-CH-UA-Bitness, Sec-CH-UA-Full-Version, Sec-CH-UA-Mobile, Sec-CH-UA-Model, Sec-CH-UA-Platform, Sec-CH-UA-Platform-Version
accept-ch: Sec-CH-UA-Arch, Sec-CH-UA-Bitness, Sec-CH-UA-Full-Version, Sec-CH-UA-Mobile, Sec-CH-UA-Model, Sec-CH-UA-Platform, Sec-CH-UA-Platform-Version
< x-msedge-ref: Ref A: 96B5C7D2C7544BEC9A9BE2963445C96F Ref B: HKBEDGE0607 Ref C: 2023-07-06T04:44:06Z
x-msedge-ref: Ref A: 96B5C7D2C7544BEC9A9BE2963445C96F Ref B: HKBEDGE0607 Ref C: 2023-07-06T04:44:06Z
< date: Thu, 06 Jul 2023 04:44:06 GMT
date: Thu, 06 Jul 2023 04:44:06 GMT

<
<html><head><title>Object moved</title></head><body>
<h2>Object moved to <a href="https://www.bing.com:443/?toWww=1&amp;redig=CA7F9D7F0A6C47FFAFA16A6589BCBB8F">here</a>.</h2>
</body></html>
* Connection #0 to host bing.com left intact
```
<del>
### Test RateLimitPolicy

#### Test Port Based Rate Limit - refer to target in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  namespace: test
  name: ratelimit-port
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: test-gw-1
  ports:
    - port: 80
      bps: 100000
EOF
```
~~

#### Test Hostname Based Rate Limit - refer to target in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  namespace: test
  name: ratelimit-hostname-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
  hostnames:
    - hostname: httptest.localhost
      config: 
        mode: Local
        backlog: 15
        requests: 100
        statTimeWindow: 60
EOF
```

#### Test Route Based Rate Limit - refer to target in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  namespace: test
  name: ratelimit-route-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
  http:
  - match:
      path:
        type: PathPrefix
        value: /bar
    config: 
      mode: Local
      backlog: 15
      requests: 100
      statTimeWindow: 60
EOF
```

#### Test Port Based Rate Limit - refer to target cross namespace

##### Create a RateLimitPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  namespace: http
  name: ratelimit-port-cross
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    namespace: test
    name: test-gw-1
  ports:
    - port: 9090
      bps: 200000
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: test
  name: ratelimit-port-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: RateLimitPolicy
      namespace: http
  to:
    - group: gateway.networking.k8s.io
      kind: Gateway
      name: test-gw-1
EOF
```

#### Test Hostname Based Rate Limit - refer to target cross namespace

##### Create a RateLimitPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  namespace: http
  name: ratelimit-hostname-http-cross
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    namespace: http-route
    name: http-cross-1
  hostnames:
    - hostname: httptest.localhost
      config: 
        mode: Local
        backlog: 25
        requests: 200
        statTimeWindow: 19
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http-route
  name: ratelimit-hostname-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: RateLimitPolicy
      namespace: http
  to:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      name: http-cross-1
EOF
```

#### Test Route Based Rate Limit - refer to target cross namespace

##### Create a RateLimitPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  namespace: http
  name: ratelimit-route-http-cross
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    namespace: http-route
    name: http-cross-1
  http:
  - match:
      path:
        type: PathPrefix
        value: /cross
    config: 
      mode: Local
      backlog: 11
      requests: 300
      statTimeWindow: 20
EOF
```

##### ReferenceGrant
If you have created the ReferenceGrant in previous step, you can skip this step. 


### Test SessionStickyPolicy - refer to target in the same namespace

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: SessionStickyPolicy
metadata:
  namespace: test
  name: session-sticky-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
  ports:
  - port: 8080
    config:
      cookieName: xxx
      expires: 600
EOF
```

### Test SessionStickyPolicy - refer to target cross namespace

#### Create a SessionStickyPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: SessionStickyPolicy
metadata:
  namespace: test
  name: session-sticky-cross
spec:
  targetRef:
    group: ""
    kind: Service
    namespace: http
    name: httpbin-cross
  ports:
  - port: 8080
    config:
      cookieName: yyy
      expires: 666
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http
  name: session-sticky-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: SessionStickyPolicy
      namespace: test
  to:
    - group: ""
      kind: Service
      name: httpbin-cross
EOF
```

### Test LoadBalancerPolicy - refer to target in the same namespace

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: LoadBalancerPolicy
metadata:
  namespace: test
  name: lb-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
  ports:
    - port: 8080
      type: HashingLoadBalancer
EOF
```

### Test LoadBalancerPolicy - refer to target cross namespace

#### Create a LoadBalancerPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: LoadBalancerPolicy
metadata:
  namespace: test
  name: lb-policy-cross
spec:
  targetRef:
    group: ""
    kind: Service
    namespace: http
    name: httpbin-cross
  ports:
    - port: 8080
      type: HashingLoadBalancer
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http
  name: lb-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: LoadBalancerPolicy
      namespace: test
  to:
    - group: ""
      kind: Service
      name: httpbin-cross
EOF
```

### Test CircuitBreakingPolicy - refer to target in the same namespace

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: CircuitBreakingPolicy
metadata:
  namespace: test
  name: circuit-breaking-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
  ports:
    - port: 8080
      config: 
        minRequestAmount: 10
        statTimeWindow: 60
        degradedTimeWindow: 60
        degradedStatusCode: 503
        degradedResponseContent: "Service Unavailable"
EOF
```

### Test CircuitBreakingPolicy - refer to target cross namespace

#### Create a CircuitBreakingPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: CircuitBreakingPolicy
metadata:
  namespace: test
  name: circuit-breaking-cross
spec:
  targetRef:
    group: ""
    kind: Service
    namespace: http
    name: httpbin-cross
  ports:
    - port: 8080
      config: 
        minRequestAmount: 11
        statTimeWindow: 61
        degradedTimeWindow: 61
        degradedStatusCode: 500
        degradedResponseContent: "Service Unavailable"
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http
  name: circuit-breaking-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: CircuitBreakingPolicy
      namespace: test
  to:
    - group: ""
      kind: Service
      name: httpbin-cross
EOF
```

### Test AccessControlPolicy

#### Test Port Based Access Control - refer to target in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  namespace: test
  name: access-control-port
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: test-gw-1
  ports:
    - port: 80
      config: 
        blacklist:
          - 10.0.0.1
          - 192.168.0.0/24
        whitelist:
          - 192.168.77.1
        enableXFF: true
        statusCode: 403
        message: "Forbidden"
EOF
```


#### Test Hostname Based Access Control - refer to target in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  namespace: test
  name: access-control-hostname-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
  hostnames:
    - hostname: httptest.localhost
      config: 
        blacklist:
          - 10.0.1.1
          - 192.168.1.0/24
        whitelist:
          - 192.168.88.1
        enableXFF: true
        statusCode: 403
        message: "Forbidden"
EOF
```

#### Test Route Based Access Control - refer to target in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  namespace: test
  name: access-control-route-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
  http:
  - match:
      path:
        type: PathPrefix
        value: /bar
    config: 
      blacklist:
        - 10.0.2.1
        - 192.168.2.0/24
      whitelist:
        - 192.168.99.1
      enableXFF: true
      statusCode: 403
      message: "Forbidden"
EOF
```


#### Test Port Based Access Control - refer to target cross namespace

##### Create a AccessControlPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  namespace: http
  name: access-control-port-cross
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    namespace: test
    name: test-gw-1
  ports:
    - port: 9090
      config: 
        blacklist:
          - 10.0.0.2
          - 192.168.1.0/24
        whitelist:
          - 192.168.66.1
        enableXFF: true
        statusCode: 403
        message: "Forbidden"
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: test
  name: access-control-port-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: AccessControlPolicy
      namespace: http
  to:
    - group: gateway.networking.k8s.io
      kind: Gateway
      name: test-gw-1
EOF
```

#### Test Hostname Based Access Control - refer to target cross namespace

##### Create a AccessControlPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  namespace: http
  name: access-control-hostname-http-cross
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    namespace: http-route
    name: http-cross-1
  hostnames:
    - hostname: httptest.localhost
      config: 
        blacklist:
          - 10.0.2.1
          - 192.168.3.0/24
        whitelist:
          - 192.168.99.1
        enableXFF: true
        statusCode: 403
        message: "Forbidden"
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http-route
  name: access-control-hostname-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: AccessControlPolicy
      namespace: http
  to:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      name: http-cross-1
EOF
```

#### Test Route Based Access Control - refer to target cross namespace

##### Create a AccessControlPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  namespace: http
  name: access-control-route-http-cross
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    namespace: http-route
    name: http-cross-1
  http:
  - match:
      path:
        type: PathPrefix
        value: /cross
    config: 
      blacklist:
        - 10.0.7.1
        - 192.168.7.0/24
      whitelist:
        - 192.168.55.1
      enableXFF: true
      statusCode: 403
      message: "Forbidden"
EOF
```

##### ReferenceGrant
If you have created the ReferenceGrant in previous step, you can skip this step.

</del>

### Test HealthCheckPolicy - refer to target in the same namespace

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha2
kind: HealthCheckPolicy
metadata:
  namespace: test
  name: health-check-policy
spec:
  targetRefs:
  - group: ""
    kind: Service
    name: httpbin
  ports:
  - port: 8080
    healthCheck: 
      interval: 10
      maxFails: 3
      failTimeout: 1
      path: /healthz
      matches:
      - statusCodes: 
        - 200
        - 201
        body: "OK"
        headers:
          - name: Content-Type
            value: application/json
EOF
```

### Test HealthCheckPolicy - refer to target cross namespace

#### Create a HealthCheckPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha2
kind: HealthCheckPolicy
metadata:
  namespace: test
  name: health-check-cross
spec:
  targetRefs:
  - group: ""
    kind: Service
    namespace: http
    name: httpbin-cross
  ports:
  - port: 8080
    healthCheck: 
      interval: 6
      maxFails: 5
      failTimeout: 10
      path: /healthz
      matches:
      - statusCodes: 
        - 400
        - 501
        body: "OK"
        headers:
          - name: Content-Type
            value: application/text
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http
  name: health-check-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: HealthCheckPolicy
      namespace: test
  to:
    - group: ""
      kind: Service
      name: httpbin-cross
EOF
```

<del>
### Test FaultInjectionPolicy

#### Test Hostname Based Fault Injection - refer to target in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: FaultInjectionPolicy
metadata:
  namespace: test
  name: fault-injection-hostname-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
  hostnames:
    - hostname: httptest.localhost
      config: 
        delay:
          percent: 50
          fixed: 10
          unit: s
EOF
```

#### Test Route Based Fault Injection - refer to target in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: FaultInjectionPolicy
metadata:
  namespace: test
  name: fault-injection-route-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
  http:
  - match:
      path:
        type: PathPrefix
        value: /bar
    config: 
      delay:
        percent: 20
        range: 
          min: 1
          max: 10
        unit: ms
EOF
```

#### Test Hostname Based Fault Injection - refer to target cross namespace

##### Create a FaultInjectionPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: FaultInjectionPolicy
metadata:
  namespace: http
  name: fault-injection-hostname-cross
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    namespace: http-route
    name: http-cross-1
  hostnames:
    - hostname: httptest.localhost
      config: 
        delay:
          percent: 60
          fixed: 15
          unit: s
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http-route
  name: fault-injection-hostname-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: FaultInjectionPolicy
      namespace: http
  to:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      name: http-cross-1
EOF
```

#### Test Route Based Fault Injection - refer to target cross namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: FaultInjectionPolicy
metadata:
  namespace: http
  name: fault-injection-route-cross
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    namespace: http-route
    name: http-cross-1
  http:
  - match:
      path:
        type: PathPrefix
        value: /cross
    config: 
      delay:
        percent: 25
        range: 
          min: 2
          max: 9
        unit: ms
EOF
```

##### ReferenceGrant
If you have created the ReferenceGrant in previous step, you can skip this step.


### Test UpstreamTLSPolicy

#### Test UpstreamTLSPolicy - refer to target and secret in the same namespace
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: UpstreamTLSPolicy
metadata:
  namespace: test
  name: upstream-tls-all-same-ns
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
  ports:
  - port: 8080
    config:
      certificateRef:
        name: https-cert
      mTLS: true
EOF
```

#### Test UpstreamTLSPolicy - refer to target in the same namespace, secret cross namespace

Delete the previous UpstreamTLSPolicy for the same service port 8080, otherwise it will be conflicted with the new one.
```shell
kubectl -n test delete upstreamtlspolicies.gateway.flomesh.io upstream-tls-all-same-ns
```

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: UpstreamTLSPolicy
metadata:
  namespace: http
  name: upstream-tls-ts-sc
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin-cross
  ports:
  - port: 8080
    config:
      certificateRef:
        namespace: test
        name: https-cert
      mTLS: false
EOF
```

#### Create a ReferenceGrant for the secret
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: test
  name: upstream-tls-secret-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: UpstreamTLSPolicy
      namespace: http
  to:
    - group: ""
      kind: Secret
      name: https-cert
EOF
```

#### Test UpstreamTLSPolicy - refer to target cross namespace, secret in the same namespace

Delete the previous UpstreamTLSPolicy for the same service port 8080, otherwise it will be conflicted with the new one.
```shell
kubectl -n http delete upstreamtlspolicies.gateway.flomesh.io upstream-tls-ts-sc
```

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: UpstreamTLSPolicy
metadata:
  namespace: test
  name: upstream-tls-tc-ss
spec:
  targetRef:
    group: ""
    kind: Service
    namespace: http
    name: httpbin-cross
  ports:
  - port: 8080
    config:
      certificateRef:
        name: https-cert
      mTLS: true
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http
  name: upstream-tls-tc-ss-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: UpstreamTLSPolicy
      namespace: test
  to:
    - group: ""
      kind: Service
      name: httpbin-cross
EOF
```

#### Test UpstreamTLSPolicy - refer to target and secret all cross namespace

Delete the previous UpstreamTLSPolicy for the same service port 8080, otherwise it will be conflicted with the new one.
```shell
kubectl -n test delete upstreamtlspolicies.gateway.flomesh.io upstream-tls-tc-ss
```

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: UpstreamTLSPolicy
metadata:
  namespace: http-route
  name: upstream-tls-all-cross
spec:
  targetRef:
    group: ""
    kind: Service
    namespace: http
    name: httpbin-cross
  ports:
  - port: 8080
    config:
      certificateRef:
        namespace: test
        name: https-cert
      mTLS: false
EOF
```

##### Create ReferenceGrants
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http
  name: upstream-tls-all-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: UpstreamTLSPolicy
      namespace: http-route
  to:
    - group: ""
      kind: Service
      name: httpbin-cross
---
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: test
  name: upstream-tls-all-cross-2
spec:
  from:
    - group: gateway.flomesh.io
      kind: UpstreamTLSPolicy
      namespace: http-route
  to:
    - group: ""
      kind: Secret
      name: https-cert
EOF
```
</del>

### Test RetryPolicy - refer to target in the same namespace

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha2
kind: RetryPolicy
metadata:
  namespace: test
  name: retry-policy
spec:
  targetRefs:
  - group: ""
    kind: Service
    name: httpbin
  ports:
  - port: 8080
    retry:
      retryOn:
        - 5xx
      numRetries: 5
      backoffBaseInterval: 2
EOF
```

### Test RetryPolicy - refer to target cross namespace

#### Create a RetryPolicy
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha2
kind: RetryPolicy
metadata:
  namespace: test
  name: retry-policy-cross
spec:
  targetRefs:
  - group: ""
    kind: Service
    namespace: http
    name: httpbin-cross
  ports:
  - port: 8080
    retry:
      retryOn:
        - "500"
      numRetries: 7
      backoffBaseInterval: 3
EOF
```

##### Create a ReferenceGrant
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  namespace: http
  name: retry-cross-1
spec:
  from:
    - group: gateway.flomesh.io
      kind: RetryPolicy
      namespace: test
  to:
    - group: ""
      kind: Service
      name: httpbin-cross
EOF
```

### Test Gateway in NodePort mode

- Create ConfigMap for configuring the Gateway
```shell
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: nodeport
  name: gateway-config
data:
  values.yaml: |
    fsm:
      gateway:
        replicas: 2
        serviceType: NodePort
        nodePorts:
          - port: 10080
            nodePort: 30080
          - port: 10443
            nodePort: 30443
EOF
```

#### Deploy Gateway
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  namespace: nodeport
  name: test-gw-2
spec:
  gatewayClassName: fsm
  listeners:
    - protocol: HTTP
      port: 10080
      name: http
    - protocol: HTTP
      port: 10443
      name: http2
  infrastructure:
    parametersRef:
      group: ""
      kind: ConfigMap
      name: gateway-config
EOF
```