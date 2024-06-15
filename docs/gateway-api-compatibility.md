# Gateway API Compatibility

This document describes which Gateway API resources FSM supports and the extent of that support.

## Summary

| Resource                            | Support Status      |
|-------------------------------------|---------------------|
| [GatewayClass](#gatewayclass)       | Partially supported |
| [Gateway](#gateway)                 | Partially supported |
| [HTTPRoute](#httproute)             | Partially supported |
| [TLSRoute](#tlsroute)               | Partially supported |
| [GRPCRoute](#grpcroute)             | Partially supported |
| [TCPRoute](#tcproute)               | Partially supported |
| [UDPRoute](#udproute)               | Partially supported |
| [ReferenceGrant](#referencegrant)   | Supported           |
| [Custom policies](#custom-policies) | Partially supported |

## Terminology

We use the following words to describe support status:
- *Supported*. The resource or field is fully supported and conformant to the Gateway API specification.
- *Partially supported*. The resource or field is supported partially or with limitations. It will become fully supported in future releases.
- *Not supported*. The resource or field is not yet supported. It will become partially or fully supported in future releases.

## Resources

Below we list the resources and the support status of their corresponding fields. 

For a description of each field, visit the [Gateway API documentation](https://gateway-api.sigs.k8s.io/references/spec/). 

### GatewayClass 

> Status: Partially supported. 

FSM supports only GatewayClass resource whose ControllerName is `flomesh.io/gateway-controller`.

Fields:
- `spec`
	- `controllerName` - supported.
	- `parametersRef` - not supported.
	- `description` - supported.
- `status`
	- `conditions` - supported.

### Gateway

> Status: Partially supported.

FSM supports only Gateway resource whose GatewayClass's ControllerName is `flomesh.io/gateway-controller`.

Fields:
- `spec`
	* `gatewayClassName` - supported.
	* `listeners`
		* `name` - supported.
		* `hostname` - supported.
		* `port` - supported. Privileged ports need to be enabled in the container security context.
		* `protocol` - supported. Allowed values: `HTTP`, `HTTPS`, `TLS`, `TCP`, `UDP`.
		* `tls`
		  * `mode` - supported. Allowed value: `Terminate`, `Passthrough`.
		  * `certificateRefs` - supported. The TLS certificate and key must be stored in a Secret of type `kubernetes.io/tls`. Multiple references are supported. You must deploy the Secrets before the Gateway resource. Secret rotation (watching for updates) is **supported**.
		  * `options` - partially supported, only ONE annotation for enable/disable mTLS.
      * `frontendValidation` - supported. Core: ConfigMap and Implementation-specific: Secret are supported. A single reference to a Kubernetes ConfigMap/Secret with the CA certificate in a key named `ca.crt`.
    * `allowedRoutes` - supported. 
	* `addresses` - not supported.
  * `infrastructure` - supported.
    * `labels` - supported. All labels are propagated to the created Deployment/Service.
    * `annotations` - supported. All annotations are propagated to the created Deployment/Service.
    * `parametersRef` - supported. Gateway configuration parameters are stored in a ConfigMap whose name must be `values.yaml` and in Helm value format defined in FSM gateway chart. Multiple references are NOT supported. You must deploy the ConfigMaps before the Gateway resource. ConfigMap rotation (watching for updates) is **supported**.
* `status`
  * `addresses` - supported.
  * `conditions` - supported, `Accepted` type for active Gateway.
  * `listeners`
	  * `name` - supported.
    * `supportedKinds` - supported.
	  * `attachedRoutes` - supported.
	  * `conditions` - supported.

### HTTPRoute

> Status: Partially supported.

Fields:
* `spec`
  * `parentRefs` - partially supported. `port` must always be set. 
  * `hostnames` - supported. 
	* `matches`
	  * `path` - supported, `Prefix`, `Exact` and `Regex`.
	  * `headers` - supported, `Exact` and `Regex`.
	  * `queryParams` - supported, `Exact` and `Regex`. 
	  * `method` -  supported.
	* `filters`
		* `type` - supported.
		* `requestRedirect`, `requestHeaderModifier`, `responseHeaderModifier`, `requestMirror`, `urlRewrite` supported 
    * `extensionRef` - not supported.
	* `backendRefs` - supported.
* `status`
  * `parents`
	* `parentRef` - supported.
	* `controllerName` - supported.
	* `conditions` - supported.


### TLSRoute

> Status: Partially supported.

Fields:
* `spec`
  * `parentRefs` - partially supported. `port` must always be set.
  * `hostnames` - supported.
  * `backendRefs` - supported.
* `status`
  * `parents`
    * `parentRef` - supported.
    * `controllerName` - supported.
    * `conditions` - supported. 

### GRPCRoute

> Status: Partially supported.

Fields:
* `spec`
  * `parentRefs` - partially supported. `port` must always be set.
  * `hostnames` - supported.
  * `matches`
    * `headers` 
      * `type` - supported, `Exact` and `Regex`.
      * `name` - supported.
      * `value` - supported.
    * method:
      * `type` - supported, `Exact` and `Regex`.
      * `service` - supported.
      * `method` -  supported.
  * `filters`
    * `type` - supported.
    * `requestHeaderModifier`, `responseHeaderModifier`, `requestMirror` supported
    * `extensionRef` - not supported.
* `status`
  * `parents`
    * `parentRef` - supported.
    * `controllerName` - supported.
    * `conditions` - supported.
      
### TCPRoute

> Status: Partially supported.

Fields:
* `spec`
  * `parentRefs` - partially supported. `port` must always be set.
  * `backendRefs` - supported.
* `status`
  * `parents`
    * `parentRef` - supported.
    * `controllerName` - supported.
    * `conditions` - supported.
      
### UDPRoute

> Status: Partially supported.

Fields:
* `spec`
  * `parentRefs` - partially supported. `port` must always be set.
  * `backendRefs` - supported.
* `status`
  * `parents`
    * `parentRef` - supported.
    * `controllerName` - supported.
    * `conditions` - supported.
    
### ReferenceGrant

> Status: supported.

### Custom Policies

> Status: Partially supported.

Custom policies will be FSM-specific CRDs that will allow supporting features like timeouts, load-balancing methods, authentication, etc. - important data-plane features that are not part of the Gateway API spec.

While those CRDs are not part of the Gateway API, the mechanism of attaching them to Gateway API resources is part of the Gateway API. See the [Policy Attachment doc](https://gateway-api.sigs.k8s.io/references/policy-attachment/).

| Policy                | Attached to Kind              | Attached Aspect                                                               | Status                                                                                          |
|-----------------------|-------------------------------|-------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| RateLimitPolicy       | Gateway, HTTPRoute, GRPCRoute | Gateway: port<br/> HTTPRoute: hostname, route<br/> GRPCRoute: hostname, route | Done.                                                                                           |
| AccessControlPolicy   | Gateway, HTTPRoute, GRPCRoute | Gateway: port<br/> HTTPRoute: hostname, route<br/> GRPCRoute: hostname, route | Done.                                                                                           |
| FaultInjectionPolicy  | HTTPRoute, GRPCRoute          | HTTPRoute: hostname, route<br/> GRPCRoute: hostname, route                    | Done.                                                                                           |
| CircuitBreakingPolicy | Service, ServiceImport        | port                                                                          | Done.                                                                                           |
| HealthCheckPolicy     | Service, ServiceImport        | port                                                                          | Done.                                                                                           |
| LoadBalancerPolicy    | Service, ServiceImport        | port                                                                          | Done.                                                                                           |
| SessionStickyPolicy   | Service, ServiceImport        | port                                                                          | Done.                                                                                           |
| RetryPolicy           | Service, ServiceImport        | port                                                                          | Done.                                                                                           |
| UpstreamTLSPolicy     | Service, ServiceImport        | port                                                                          | Partially done, only support service port level TLS config, expect to control at endpoint level |


## Listener Protocol and Supported Route Types

| Listener Protocol | TLS Mode    | Route Type Supported |
|-------------------|-------------|----------------------|
| HTTP              |             | HTTPRoute, GRPCRoute |
| HTTPS             |             | HTTPRoute, GRPCRoute |
| TLS               | Passthrough | TLSRoute             |
| TLS               | Terminate   | TCPRoute             |
| TCP               |             | TCPRoute             |
| UDP               |             | UDPRoute             |