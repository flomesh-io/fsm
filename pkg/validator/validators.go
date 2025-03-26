package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	mapset "github.com/deckarep/golang-set"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	admissionv1 "k8s.io/api/admission/v1"

	"k8s.io/apimachinery/pkg/util/validation/field"

	pluginv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/plugin/v1alpha1"
	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/service"
)

// validateFunc is a function type that accepts an AdmissionRequest and returns an AdmissionResponse.
/*
There are a few ways to utilize the Validator function:

1. return resp, nil

	In this case we simply return the raw resp. This allows for the most customization.

2. return nil, err

	In this case we convert the error to an AdmissionResponse.  If the error type is an AdmissionError, we
	convert accordingly, which allows for some customization of the AdmissionResponse. Otherwise, we set Allow to
	false and the status to the error message.

3. return nil, nil

	In this case we create a simple AdmissionResponse, with Allow set to true.

4. Note that resp, err will ignore the error. It assumes that you are returning nil for resp if there is an error

In all of the above cases we always populate the UID of the response from the request.

An example of a validator:

func FakeValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	o, n := &FakeObj{}, &FakeObj{}
	// If you need to compare against the old object
	if err := json.NewDecoder(bytes.NewBuffer(req.OldObject.Raw)).Decode(o); err != nil {
		return nil, err
	}

	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(n); err != nil {
		returrn nil, err
	}

	// validate the objects, potentially returning an error, or a more detailed AdmissionResponse.

	// This will set allow to true
	return nil, nil
}
*/
type validateFunc func(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error)

// policyValidator is a validator that has access to a policy
type policyValidator struct {
	policyClient   policy.Controller
	kubeController k8s.Controller
	cfg            *configurator.Client
}

func trafficTargetValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	trafficTarget := &smiAccess.TrafficTarget{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(trafficTarget); err != nil {
		return nil, err
	}

	if trafficTarget.Spec.Destination.Namespace != trafficTarget.Namespace {
		return nil, fmt.Errorf("The traffic target namespace (%s) must match spec.Destination.Namespace (%s)",
			trafficTarget.Namespace, trafficTarget.Spec.Destination.Namespace)
	}

	return nil, nil
}

// accessCertValidator validates the AccessCert custom resource
func (kc *policyValidator) accessCertValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	act := &policyv1alpha1.AccessCert{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(act); err != nil {
		return nil, err
	}
	if !kc.cfg.GetFeatureFlags().EnableAccessCertPolicy {
		return nil, fmt.Errorf("FSM is prohibited to issue certificates for external services")
	}
	return nil, nil
}

// accessControlValidator validates the AccessControl custom resource
func (kc *policyValidator) accessControlValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	acl := &policyv1alpha1.AccessControl{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(acl); err != nil {
		return nil, err
	}
	ns := acl.Namespace

	type setEntry struct {
		name string
		port int
	}

	backends := mapset.NewSet()
	var conflictString strings.Builder
	conflictingAcls := mapset.NewSet()
	for _, backend := range acl.Spec.Backends {
		if unique := backends.Add(setEntry{backend.Name, backend.Port.Number}); !unique {
			return nil, fmt.Errorf("Duplicate backends detected with service name: %s and port: %d", backend.Name, backend.Port.Number)
		}

		fakeMeshSvc := service.MeshService{
			Name:       backend.Name,
			TargetPort: uint16(backend.Port.Number),
			Protocol:   backend.Port.Protocol,
		}

		if matchingPolicy := kc.policyClient.GetAccessControlPolicy(fakeMeshSvc); matchingPolicy != nil && matchingPolicy.Name != acl.Name {
			// we've found a duplicate
			if unique := conflictingAcls.Add(matchingPolicy); !unique {
				// we've already found the conflicts for this resource
				continue
			}
			conflicts := policy.DetectAccessControlConflicts(*acl, *matchingPolicy)
			fmt.Fprintf(&conflictString, "[+] AccessControlBackend %s/%s conflicts with %s/%s:\n", ns, acl.GetName(), ns, matchingPolicy.GetName())
			for _, err := range conflicts {
				fmt.Fprintf(&conflictString, "%s\n", err)
			}
			fmt.Fprintf(&conflictString, "\n")
		}

		if backend.TLS != nil {
			// If mTLS is enabled, verify there is an AuthenticatedPrincipal specified
			authenticatedSourceFound := false
			for _, source := range acl.Spec.Sources {
				if source.Kind == policyv1alpha1.KindAuthenticatedPrincipal {
					authenticatedSourceFound = true
					break
				}
			}

			if backend.TLS.SkipClientCertValidation && !authenticatedSourceFound {
				return nil, fmt.Errorf("HTTPS acl with client certificate validation enabled must specify at least one 'AuthenticatedPrincipal` source")
			}
		}
	}

	if conflictString.Len() != 0 {
		return nil, fmt.Errorf("duplicate backends detected\n%s", conflictString.String())
	}

	// Validate sources
	for _, source := range acl.Spec.Sources {
		switch source.Kind {
		// Add validation for source kinds here
		case policyv1alpha1.KindService:
			if source.Name == "" {
				return nil, fmt.Errorf("'source.name' not specified for source kind %s", policyv1alpha1.KindService)
			}
			if source.Namespace == "" {
				return nil, fmt.Errorf("'source.namespace' not specified for source kind %s", policyv1alpha1.KindService)
			}

		case policyv1alpha1.KindAuthenticatedPrincipal:
			if source.Name == "" {
				return nil, fmt.Errorf("'source.name' not specified for source kind %s", policyv1alpha1.KindAuthenticatedPrincipal)
			}

		case policyv1alpha1.KindIPRange:
			if _, _, err := net.ParseCIDR(source.Name); err != nil {
				return nil, fmt.Errorf("Invalid 'source.name' value specified for IPRange. Expected CIDR notation 'a.b.c.d/x', got '%s'", source.Name)
			}

		default:
			return nil, fmt.Errorf("Invalid 'source.kind' value specified. Must be one of: %s, %s, %s",
				policyv1alpha1.KindService, policyv1alpha1.KindAuthenticatedPrincipal, policyv1alpha1.KindIPRange)
		}
	}

	return nil, nil
}

// ingressBackendValidator validates the IngressBackend custom resource
func (kc *policyValidator) ingressBackendValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	ingressBackend := &policyv1alpha1.IngressBackend{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(ingressBackend); err != nil {
		return nil, err
	}
	ns := ingressBackend.Namespace

	type setEntry struct {
		name string
		port int
	}

	backends := mapset.NewSet()
	var conflictString strings.Builder
	conflictingIngressBackends := mapset.NewSet()
	for _, backend := range ingressBackend.Spec.Backends {
		if unique := backends.Add(setEntry{backend.Name, backend.Port.Number}); !unique {
			return nil, fmt.Errorf("Duplicate backends detected with service name: %s and port: %d", backend.Name, backend.Port.Number)
		}

		fakeMeshSvc := service.MeshService{
			Name:       backend.Name,
			TargetPort: uint16(backend.Port.Number),
			Protocol:   backend.Port.Protocol,
		}

		if matchingPolicy := kc.policyClient.GetIngressBackendPolicy(fakeMeshSvc); matchingPolicy != nil && matchingPolicy.Name != ingressBackend.Name {
			// we've found a duplicate
			if unique := conflictingIngressBackends.Add(matchingPolicy); !unique {
				// we've already found the conflicts for this resource
				continue
			}
			conflicts := policy.DetectIngressBackendConflicts(*ingressBackend, *matchingPolicy)
			fmt.Fprintf(&conflictString, "[+] IngressBackend %s/%s conflicts with %s/%s:\n", ns, ingressBackend.GetName(), ns, matchingPolicy.GetName())
			for _, err := range conflicts {
				fmt.Fprintf(&conflictString, "%s\n", err)
			}
			fmt.Fprintf(&conflictString, "\n")
		}

		// Validate port
		switch strings.ToLower(backend.Port.Protocol) {
		case constants.ProtocolHTTP:
			// Valid
		case constants.ProtocolHTTPS:
			// Valid
		case constants.ProtocolGRPC:
			// Valid
			// If mTLS is enabled, verify there is an AuthenticatedPrincipal specified
			authenticatedSourceFound := false
			for _, source := range ingressBackend.Spec.Sources {
				if source.Kind == policyv1alpha1.KindAuthenticatedPrincipal {
					authenticatedSourceFound = true
					break
				}
			}

			if backend.TLS != nil && backend.TLS.SkipClientCertValidation && !authenticatedSourceFound {
				return nil, fmt.Errorf("HTTPS ingress with client certificate validation enabled must specify at least one 'AuthenticatedPrincipal` source")
			}

		default:
			return nil, fmt.Errorf("Expected 'port.protocol' to be 'http', 'https' or 'grpc', got: %s", backend.Port.Protocol)
		}
	}

	if conflictString.Len() != 0 {
		return nil, fmt.Errorf("duplicate backends detected\n%s", conflictString.String())
	}

	// Validate sources
	for _, source := range ingressBackend.Spec.Sources {
		switch source.Kind {
		// Add validation for source kinds here
		case policyv1alpha1.KindService:
			if source.Name == "" {
				return nil, fmt.Errorf("'source.name' not specified for source kind %s", policyv1alpha1.KindService)
			}
			if source.Namespace == "" {
				return nil, fmt.Errorf("'source.namespace' not specified for source kind %s", policyv1alpha1.KindService)
			}

		case policyv1alpha1.KindAuthenticatedPrincipal:
			if source.Name == "" {
				return nil, fmt.Errorf("'source.name' not specified for source kind %s", policyv1alpha1.KindAuthenticatedPrincipal)
			}

		case policyv1alpha1.KindIPRange:
			if _, _, err := net.ParseCIDR(source.Name); err != nil {
				return nil, fmt.Errorf("Invalid 'source.name' value specified for IPRange. Expected CIDR notation 'a.b.c.d/x', got '%s'", source.Name)
			}

		default:
			return nil, fmt.Errorf("Invalid 'source.kind' value specified. Must be one of: %s, %s, %s",
				policyv1alpha1.KindService, policyv1alpha1.KindAuthenticatedPrincipal, policyv1alpha1.KindIPRange)
		}
	}

	return nil, nil
}

// egressValidator validates the Egress custom resource
func egressValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	egress := &policyv1alpha1.Egress{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(egress); err != nil {
		return nil, err
	}

	// Validate match references
	allowedAPIGroups := []string{smiSpecs.SchemeGroupVersion.String(), policyv1alpha1.SchemeGroupVersion.String()}
	upstreamTrafficSettingMatchCount := 0
	for _, m := range egress.Spec.Matches {
		switch *m.APIGroup {
		case smiSpecs.SchemeGroupVersion.String():
			switch m.Kind {
			case "HTTPRouteGroup":
				// no additional validation

			default:
				return nil, fmt.Errorf("Expected 'matches.kind' for match '%s' to be 'HTTPRouteGroup', got: %s", m.Name, m.Kind)
			}

		case policyv1alpha1.SchemeGroupVersion.String():
			switch m.Kind {
			case "UpstreamTrafficSetting":
				upstreamTrafficSettingMatchCount++

			default:
				return nil, fmt.Errorf("Expected 'matches.kind' for match '%s' to be 'UpstreamTrafficSetting', got: %s", m.Name, m.Kind)
			}

		default:
			return nil, fmt.Errorf("Expected 'matches.apiGroup' to be one of %v, got: %s", allowedAPIGroups, *m.APIGroup)
		}
	}

	// Can't have more than 1 UpstreamTrafficSetting match for an Egress policy
	if upstreamTrafficSettingMatchCount > 1 {
		return nil, fmt.Errorf("Cannot have more than 1 UpstreamTrafficSetting match")
	}

	return nil, nil
}

// upstreamTrafficSettingValidator validates the UpstreamTrafficSetting custom resource
func (kc *policyValidator) upstreamTrafficSettingValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	upstreamTrafficSetting := &policyv1alpha1.UpstreamTrafficSetting{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(upstreamTrafficSetting); err != nil {
		return nil, err
	}

	ns := upstreamTrafficSetting.Namespace
	hostComponents := strings.Split(upstreamTrafficSetting.Spec.Host, ".")
	if len(hostComponents) < 2 {
		return nil, field.Invalid(field.NewPath("spec").Child("host"), upstreamTrafficSetting.Spec.Host, "invalid FQDN specified as host")
	}

	opt := policy.UpstreamTrafficSettingGetOpt{Host: upstreamTrafficSetting.Spec.Host}
	if matchingUpstreamTrafficSetting := kc.policyClient.GetUpstreamTrafficSetting(opt); matchingUpstreamTrafficSetting != nil && matchingUpstreamTrafficSetting.Name != upstreamTrafficSetting.Name {
		// duplicate detected
		return nil, fmt.Errorf("UpstreamTrafficSetting %s/%s conflicts with %s/%s since they have the same host %s", ns, upstreamTrafficSetting.GetName(), ns, matchingUpstreamTrafficSetting.GetName(), matchingUpstreamTrafficSetting.Spec.Host)
	}

	// Validate rate limiting config
	rl := upstreamTrafficSetting.Spec.RateLimit
	if rl != nil && rl.Local != nil && rl.Local.HTTP != nil {
		if _, ok := statusCodeName[int32(rl.Local.HTTP.ResponseStatusCode)]; !ok {
			return nil, fmt.Errorf("invalid responseStatusCode %d",
				rl.Local.HTTP.ResponseStatusCode)
		}
	}
	for _, route := range upstreamTrafficSetting.Spec.HTTPRoutes {
		if route.RateLimit != nil && route.RateLimit.Local != nil {
			if _, ok := statusCodeName[int32(route.RateLimit.Local.ResponseStatusCode)]; !ok {
				return nil, fmt.Errorf("invalid responseStatusCode %d",
					route.RateLimit.Local.ResponseStatusCode)
			}
		}
	}

	return nil, nil
}

// egressGatewayValidator validates the EgressGateway custom resource
func (kc *policyValidator) egressGatewayValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	egressGateway := &policyv1alpha1.EgressGateway{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(egressGateway); err != nil {
		return nil, err
	}

	if len(egressGateway.Spec.GlobalEgressGateways) > 0 {
		existEgressGateways := kc.policyClient.ListEgressGateways()
		for _, p := range existEgressGateways {
			if strings.EqualFold(egressGateway.Name, p.Name) && strings.EqualFold(egressGateway.Namespace, p.Namespace) {
				continue
			}
			if len(p.Spec.GlobalEgressGateways) > 0 {
				return nil, fmt.Errorf("Redefinition of global egress gateway policy and conflict with %s.%s", p.Namespace, p.Name)
			}
		}
	}

	return nil, nil
}

// pluginValidator validates the plugin custom resource
func (kc *policyValidator) pluginValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	if !kc.cfg.GetFeatureFlags().EnablePluginPolicy {
		return nil, fmt.Errorf("FSM is prohibited to apply plugin policy")
	}

	plugin := &pluginv1alpha1.Plugin{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(plugin); err != nil {
		return nil, err
	}

	if len(plugin.Spec.Script) == 0 {
		return nil, fmt.Errorf("plugin[%s.%s] is missing pipy script", plugin.Name, plugin.Namespace)
	}

	return nil, nil
}

// pluginConfigValidator validates the plugin config custom resource
func (kc *policyValidator) pluginConfigValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	if !kc.cfg.GetFeatureFlags().EnablePluginPolicy {
		return nil, fmt.Errorf("FSM is prohibited to apply plugin policy")
	}

	pluginConfig := &pluginv1alpha1.PluginConfig{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(pluginConfig); err != nil {
		return nil, err
	}

	return nil, nil
}

// pluginChainValidator validates the plugin chain custom resource
func (kc *policyValidator) pluginChainValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	if !kc.cfg.GetFeatureFlags().EnablePluginPolicy {
		return nil, fmt.Errorf("FSM is prohibited to apply plugin policy")
	}

	pluginChain := &pluginv1alpha1.PluginChain{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(pluginChain); err != nil {
		return nil, err
	}

	return nil, nil
}

var (
	statusCodeName = map[int32]string{
		0:   "Empty",
		100: "Continue",
		200: "OK",
		201: "Created",
		202: "Accepted",
		203: "NonAuthoritativeInformation",
		204: "NoContent",
		205: "ResetContent",
		206: "PartialContent",
		207: "MultiStatus",
		208: "AlreadyReported",
		226: "IMUsed",
		300: "MultipleChoices",
		301: "MovedPermanently",
		302: "Found",
		303: "SeeOther",
		304: "NotModified",
		305: "UseProxy",
		307: "TemporaryRedirect",
		308: "PermanentRedirect",
		400: "BadRequest",
		401: "Unauthorized",
		402: "PaymentRequired",
		403: "Forbidden",
		404: "NotFound",
		405: "MethodNotAllowed",
		406: "NotAcceptable",
		407: "ProxyAuthenticationRequired",
		408: "RequestTimeout",
		409: "Conflict",
		410: "Gone",
		411: "LengthRequired",
		412: "PreconditionFailed",
		413: "PayloadTooLarge",
		414: "URITooLong",
		415: "UnsupportedMediaType",
		416: "RangeNotSatisfiable",
		417: "ExpectationFailed",
		421: "MisdirectedRequest",
		422: "UnprocessableEntity",
		423: "Locked",
		424: "FailedDependency",
		426: "UpgradeRequired",
		428: "PreconditionRequired",
		429: "TooManyRequests",
		431: "RequestHeaderFieldsTooLarge",
		500: "InternalServerError",
		501: "NotImplemented",
		502: "BadGateway",
		503: "ServiceUnavailable",
		504: "GatewayTimeout",
		505: "HTTPVersionNotSupported",
		506: "VariantAlsoNegotiates",
		507: "InsufficientStorage",
		508: "LoopDetected",
		510: "NotExtended",
		511: "NetworkAuthenticationRequired",
	}
)
