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
| [UDPRoute](#udproute)               | Not supported       |
| [ReferencePolicy](#referencepolicy) | Not supported       |
| [ReferenceGrant](#referencegrant)   | Not supported       |
| [Custom policies](#custom-policies) | Not supported       |

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

FSM supports only GatewayClass resource whose ControllerName is `flomesh.io/gateway-controller`. If multiple valid GatewayClasses are created, the oldest is active and take effect.


Fields:
- `spec`
	- `controllerName` - supported.
	- `parametersRef` - not supported.
	- `description` - supported.
- `status`
	- `conditions` - partially supported. Support `Accepted` type and added ConditionType `Active`.

### Gateway

> Status: Partially supported.

FSM supports only a single Gateway resource per namespace. 
The Gateway resource must reference FSM's corresponding effective GatewayClass. 
In case of multiple Gateway resources created in the same namespace, FSM will choose the oldest ONE by creation timestamp. If the timestamps are equal, FSM will choose the resource that appears first in alphabetical order by “{name}”. We might support multiple Gateway resources. 

Fields:
- `spec`
	* `gatewayClassName` - supported.
	* `listeners`
		* `name` - supported.
		* `hostname` - supported.
		* `port` - supported, must be LTE 60000, all priviliged ports will be mapped to 60000 + port.
		* `protocol` - partially supported. Allowed values: `HTTP`, `HTTPS`, `TLS`, `TCP`.
		* `tls`
		  * `mode` - partially supported. Allowed value: `Terminate`.
		  * `certificateRefs` - partially supported. The TLS certificate and key must be stored in a Secret resource of type `kubernetes.io/tls`. Multiple references are supported. You must deploy the Secrets before the Gateway resource. Secret rotation (watching for updates) is not supported.
		  * `options` - not supported.
		* `allowedRoutes` - not supported. 
	* `addresses` - not supported.
* `status`
  * `addresses` - supported.
  * `conditions` - supported, `Accepted` type for active Gateway.
  * `listeners`
	* `name` - supported.
	* `supportedKinds` - not supported.
	* `attachedRoutes` - supported.
	* `conditions` - partially supported.

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
	* `conditions` - partially supported. Supported (Condition/Status/Reason):
    	*  `Accepted/True/Accepted`
    	*  `Accepted/False/NoMatchingListenerHostname`


### TLSRoute

> Status: Partially supported.

Fields:
* `spec`
  * `parentRefs` - partially supported. `port` must always be set.
  * `hostnames` - supported.
  * `backendRefs` - supported.


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

### TCPRoute

> Status: Partially supported.

Fields:
* `spec`
  * `parentRefs` - partially supported. `port` must always be set.
  * `backendRefs` - supported.

### UDPRoute

> Status: Not supported.

### ReferencePolicy

> Status: Not supported(Officially Deprecated).
 
### ReferenceGrant

> Status: Not supported.

### Custom Policies

> Status: Not supported.

Custom policies will be FSM-specific CRDs that will allow supporting features like timeouts, load-balancing methods, authentication, etc. - important data-plane features that are not part of the Gateway API spec.

While those CRDs are not part of the Gateway API, the mechanism of attaching them to Gateway API resources is part of the Gateway API. See the [Policy Attachment doc](https://gateway-api.sigs.k8s.io/references/policy-attachment/).

## Listener Protocol and Supported Route Types

| Listener Protocol | TLS Mode    | Route Type Supported |
|-------------------|-------------|----------------------|
| HTTP              |             | HTTPRoute, GRPCRoute |
| HTTPS             |             | HTTPRoute, GRPCRoute |
| TLS               | Passthrough | TLSRoute             |
| TLS               | Terminate   | TCPRoute             |
| TCP               |             | TCPRoute             |