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
  - `7443`: **HTTPS**
  - `8443`: **TLS Passthrough**
  - `9443`: **TLS Terminate**
  - `3000`: **TCP**
  - `4000`: **UDP**
  
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
kubectl create ns httpbin
kubectl create ns grpcbin
kubectl create ns tcproute
kubectl create ns udproute
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
kubectl -n httpbin create secret generic https-cert \
  --from-file=ca.crt=./ca.crt \
  --from-file=tls.crt=./https.crt \
  --from-file=tls.key=./https.key 
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
kubectl -n grpcbin create secret tls grpc-cert --key grpc.key --cert grpc.crt
```

#### Deploy Gateway
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: test-gw-1
  annotations:
    gateway.flomesh.io/replicas: "2"
    gateway.flomesh.io/cpu: 100m
    gateway.flomesh.io/cpu-limit: 1000m
    gateway.flomesh.io/memory: 256Mi
    gateway.flomesh.io/memory-limit: 1024Mi
spec:
  gatewayClassName: fsm-gateway-cls
  listeners:
    - protocol: HTTP
      port: 80
      name: http
    - protocol: TCP
      port: 3000
      name: tcp
    - protocol: HTTPS
      port: 443
      name: https
      tls:
        certificateRefs:
          - name: https-cert
            namespace: httpbin
          - name: grpc-cert
            namespace: grpcbin
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
            namespace: httpbin
          - name: grpc-cert
            namespace: grpcbin
    - protocol: UDP
      port: 4000
      name: udp
  infrastructure:
    annotations:
      xyz: abc
    labels:
      test: demo
EOF
```

### Test HTTPRoute

#### Deploy a HTTP Service
```shell
kubectl -n httpbin apply -f - <<EOF
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
    app: pipy
---
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
          image: flomesh/pipy:0.99.1-1
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
EOF
```

#### Create a HTTPRoute
```shell
kubectl -n httpbin apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: http-app-1
spec:
  parentRefs:
  - name: test-gw-1
    namespace: default
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
*   Trying 127.0.0.1:8090...
* Connected to localhost (127.0.0.1) port 8090 (#0)
> GET /bar HTTP/1.1
> Host: httptest.localhost
> User-Agent: curl/7.88.1
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
* Connection #0 to host localhost left intact
```

### Test GRPCRoute
#### Step 1: Create a Kubernetes `Deployment` for gRPC app

- Deploy the gRPC app

```shell
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: grpcbin
  namespace: grpcbin
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
  namespace: grpcbin
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
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: GRPCRoute
metadata:
  name: grpc-app-1
  namespace: grpcbin
spec:
  parentRefs:
    - name: test-gw-1
      namespace: default
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

### Test TCPRoute

#### Create the namespace

#### Deploy the TCPRoute app
```shell
kubectl -n tcproute apply -f - <<EOF
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
kubectl -n tcproute apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: tcp-app-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: default
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

### Test UDPRoute

#### Deploy the UDPRoute app
```shell
kubectl -n udproute apply -f - <<EOF
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
kubectl -n udproute apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: UDPRoute
metadata:
  name: udp-app-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: default
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

### Test HTTPS - HTTPRoute
#### Create a HTTPRoute and attach to HTTPS port
```shell
kubectl -n httpbin apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: https-app-1
spec:
  parentRefs:
  - name: test-gw-1
    namespace: default
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
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: GRPCRoute
metadata:
  name: grpcs-app-1
  namespace: grpcbin
spec:
  parentRefs:
    - name: test-gw-1
      namespace: default
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
kubectl -n tcproute apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: tlst-app-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: default
      port: 9443
  rules:
  - backendRefs:
    - name: tcproute
      port: 8080
EOF
```

#### Test it:
```shell
❯ curl -iv --cacert https.crt https://httptest.localhost:9443
*   Trying 127.0.0.1:9443...
* Connected to httptest.localhost (127.0.0.1) port 9443 (#0)
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
* h2h3 [:path: /]
* h2h3 [:scheme: https]
* h2h3 [:authority: httptest.localhost]
* h2h3 [user-agent: curl/7.88.1]
* h2h3 [accept: */*]
* Using Stream ID: 1 (easy handle 0x7f90ec011e00)
> GET / HTTP/2
> Host: httptest.localhost
> user-agent: curl/7.88.1
> accept: */*
>
< HTTP/2 200
HTTP/2 200

<
Hi, I am TCPRoute!
* Connection #0 to host httptest.localhost left intact
```

### Test TLS Passthrough

#### Create TLSRoute
```shell
kubectl -n tcproute apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TLSRoute
metadata:
  name: tlsp-app-1
spec:
  parentRefs:
    - name: test-gw-1
      namespace: default
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

### Test RateLimitPolicy

#### Test Port Based Rate Limit
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  name: ratelimit-port
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: test-gw-1
    namespace: default
  ports:
    - port: 80
      bps: 100000
EOF
```


#### Test Hostname Based Rate Limit
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  name: ratelimit-hostname-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
    namespace: httpbin
  hostnames:
    - hostname: httptest.localhost
      config: 
        mode: Local
        backlog: 15
        requests: 100
        statTimeWindow: 60
EOF
```

#### Test Route Based Rate Limit
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RateLimitPolicy
metadata:
  name: ratelimit-route-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
    namespace: httpbin
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

### Test SessionStickyPolicy

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: SessionStickyPolicy
metadata:
  name: session-sticky-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
    namespace: httpbin
  ports:
  - port: 8080
    config:
      cookieName: xxx
      expires: 600
EOF
```

### Test LoadBalancerPolicy

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: LoadBalancerPolicy
metadata:
  name: lb-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
    namespace: httpbin
  ports:
    - port: 8080
      type: HashingLoadBalancer
EOF
```

### Test CircuitBreakingPolicy

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: CircuitBreakingPolicy
metadata:
  name: circuit-breaking-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
    namespace: httpbin
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

### Test AccessControlPolicy

#### Test Port Based Access Control
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  name: access-control-port
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: test-gw-1
    namespace: default
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


#### Test Hostname Based Access Control
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  name: access-control-hostname-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
    namespace: httpbin
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

#### Test Route Based Access Control
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: AccessControlPolicy
metadata:
  name: access-control-route-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
    namespace: httpbin
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

### Test HealthCheckPolicy

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: HealthCheckPolicy
metadata:
  name: health-check-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
    namespace: httpbin
  ports:
  - port: 8080
    config: 
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

### Test FaultInjectionPolicy


#### Test Hostname Based Fault Injection
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: FaultInjectionPolicy
metadata:
  name: fault-injection-hostname-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
    namespace: httpbin
  hostnames:
    - hostname: httptest.localhost
      config: 
        delay:
          percent: 50
          fixed: 10
          unit: s
EOF
```

#### Test Route Based Fault Injection
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: FaultInjectionPolicy
metadata:
  name: fault-injection-route-http
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: http-app-1
    namespace: httpbin
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

### Test UpstreamTLSPolicy

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: UpstreamTLSPolicy
metadata:
  name: upstream-tls-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
    namespace: httpbin
  ports:
  - port: 8080
    config:
      certificateRef:
        namespace: httpbin
        name: https-cert
      mTLS: false
EOF
```

### Test RetryPolicy

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: RetryPolicy
metadata:
  name: retry-policy
spec:
  targetRef:
    group: ""
    kind: Service
    name: httpbin
    namespace: httpbin
  ports:
  - port: 8080
    config:
      retryOn:
        - 5xx
      numRetries: 5
      backoffBaseInterval: 2
EOF
```

### Test GatewayTLSPolicy

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.flomesh.io/v1alpha1
kind: GatewayTLSPolicy
metadata:
  name: gateway-tls-policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: test-gw-1
    namespace: default
  ports:
  - port: 443
    config:
      mTLS: true
EOF
```