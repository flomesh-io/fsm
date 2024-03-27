package cache

import (
	"fmt"
	"reflect"

	"github.com/flomesh-io/fsm/pkg/utils"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"golang.org/x/exp/slices"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

func isRefToService(ref gwv1.BackendObjectReference, service client.ObjectKey, ns string) bool {
	if ref.Group == nil {
		return false
	}

	if ref.Kind == nil {
		return false
	}

	if (string(*ref.Group) == constants.KubernetesCoreGroup && string(*ref.Kind) == constants.KubernetesServiceKind) ||
		(string(*ref.Group) == constants.FlomeshAPIGroup && string(*ref.Kind) == constants.FlomeshAPIServiceImportKind) {
		if ref.Namespace == nil {
			if ns != service.Namespace {
				return false
			}
		} else {
			if string(*ref.Namespace) != service.Namespace {
				return false
			}
		}

		return string(ref.Name) == service.Name
	}

	return false
}

func isRefToSecret(ref gwv1.SecretObjectReference, secret client.ObjectKey, ns string) bool {
	if ref.Group == nil {
		return false
	}

	if ref.Kind == nil {
		return false
	}

	if string(*ref.Group) == constants.KubernetesCoreGroup && string(*ref.Kind) == constants.KubernetesSecretKind {
		if ref.Namespace == nil {
			if ns != secret.Namespace {
				return false
			}
		} else {
			if string(*ref.Namespace) != secret.Namespace {
				return false
			}
		}

		return string(ref.Name) == secret.Name
	}

	return false
}

func allowedListeners(
	parentRef gwv1.ParentReference,
	routeGvk schema.GroupVersionKind,
	validListeners []gwtypes.Listener,
) []gwtypes.Listener {
	var selectedListeners []gwtypes.Listener
	for _, validListener := range validListeners {
		if (parentRef.SectionName == nil || *parentRef.SectionName == validListener.Name) &&
			(parentRef.Port == nil || *parentRef.Port == validListener.Port) {
			selectedListeners = append(selectedListeners, validListener)
		}
	}
	log.Debug().Msgf("[GW-CACHE] selectedListeners: %v", selectedListeners)

	if len(selectedListeners) == 0 {
		return nil
	}

	allowedListeners := make([]gwtypes.Listener, 0)
	for _, selectedListener := range selectedListeners {
		if !selectedListener.AllowsKind(routeGvk) {
			continue
		}

		allowedListeners = append(allowedListeners, selectedListener)
	}

	if len(allowedListeners) == 0 {
		return nil
	}

	return allowedListeners
}

func httpPathMatchType(matchType *gwv1.PathMatchType) fgw.MatchType {
	if matchType == nil {
		return fgw.MatchTypePrefix
	}

	switch *matchType {
	case gwv1.PathMatchPathPrefix:
		return fgw.MatchTypePrefix
	case gwv1.PathMatchExact:
		return fgw.MatchTypeExact
	case gwv1.PathMatchRegularExpression:
		return fgw.MatchTypeRegex
	default:
		return fgw.MatchTypePrefix
	}
}

func httpPath(value *string) string {
	if value == nil {
		return "/"
	}

	return *value
}

func httpMatchHeaders(m gwv1.HTTPRouteMatch) map[fgw.MatchType]map[string]string {
	exact := make(map[string]string)
	regex := make(map[string]string)

	for _, header := range m.Headers {
		if header.Type == nil {
			exact[string(header.Name)] = header.Value
			continue
		}

		switch *header.Type {
		case gwv1.HeaderMatchExact:
			exact[string(header.Name)] = header.Value
		case gwv1.HeaderMatchRegularExpression:
			regex[string(header.Name)] = header.Value
		}
	}

	headers := make(map[fgw.MatchType]map[string]string)

	if len(exact) > 0 {
		headers[fgw.MatchTypeExact] = exact
	}

	if len(regex) > 0 {
		headers[fgw.MatchTypeRegex] = regex
	}

	return headers
}

func httpMatchQueryParams(m gwv1.HTTPRouteMatch) map[fgw.MatchType]map[string]string {
	exact := make(map[string]string)
	regex := make(map[string]string)

	for _, param := range m.QueryParams {
		if param.Type == nil {
			exact[string(param.Name)] = param.Value
			continue
		}

		switch *param.Type {
		case gwv1.QueryParamMatchExact:
			exact[string(param.Name)] = param.Value
		case gwv1.QueryParamMatchRegularExpression:
			regex[string(param.Name)] = param.Value
		}
	}

	params := make(map[fgw.MatchType]map[string]string)
	if len(exact) > 0 {
		params[fgw.MatchTypeExact] = exact
	}

	if len(regex) > 0 {
		params[fgw.MatchTypeRegex] = regex
	}

	return params
}

func grpcMethodMatchType(matchType *gwv1alpha2.GRPCMethodMatchType) fgw.MatchType {
	if matchType == nil {
		return fgw.MatchTypeExact
	}

	switch *matchType {
	case gwv1alpha2.GRPCMethodMatchExact:
		return fgw.MatchTypeExact
	case gwv1alpha2.GRPCMethodMatchRegularExpression:
		return fgw.MatchTypeRegex
	default:
		return fgw.MatchTypeExact
	}
}

func grpcMatchHeaders(m gwv1alpha2.GRPCRouteMatch) map[fgw.MatchType]map[string]string {
	exact := make(map[string]string)
	regex := make(map[string]string)

	for _, header := range m.Headers {
		if header.Type == nil {
			exact[string(header.Name)] = header.Value
			continue
		}

		switch *header.Type {
		case gwv1.HeaderMatchExact:
			exact[string(header.Name)] = header.Value
		case gwv1.HeaderMatchRegularExpression:
			regex[string(header.Name)] = header.Value
		}
	}

	headers := make(map[fgw.MatchType]map[string]string)

	if len(exact) > 0 {
		headers[fgw.MatchTypeExact] = exact
	}

	if len(regex) > 0 {
		headers[fgw.MatchTypeRegex] = regex
	}

	return headers
}

func backendRefToServicePortName(ref gwv1.BackendObjectReference, defaultNs string) *fgw.ServicePortName {
	// ONLY supports Service and ServiceImport backend now
	if (*ref.Kind == constants.KubernetesServiceKind && *ref.Group == constants.KubernetesCoreGroup) ||
		(*ref.Kind == constants.FlomeshAPIServiceImportKind && *ref.Group == constants.FlomeshAPIGroup) {
		return &fgw.ServicePortName{
			NamespacedName: types.NamespacedName{
				Namespace: gwutils.Namespace(ref.Namespace, defaultNs),
				Name:      string(ref.Name),
			},
			Port: pointer.Int32(int32(*ref.Port)),
		}
	}

	return nil
}

func targetRefToServicePortName(ref gwv1alpha2.PolicyTargetReference, defaultNs string, port int32) *fgw.ServicePortName {
	// ONLY supports Service and ServiceImport backend now
	if (ref.Kind == constants.KubernetesServiceKind && ref.Group == constants.KubernetesCoreGroup) ||
		(ref.Kind == constants.FlomeshAPIServiceImportKind && ref.Group == constants.FlomeshAPIGroup) {
		return &fgw.ServicePortName{
			NamespacedName: types.NamespacedName{
				Namespace: gwutils.Namespace(ref.Namespace, defaultNs),
				Name:      string(ref.Name),
			},
			Port: pointer.Int32(port),
		}
	}

	return nil
}

func passthroughTarget(ref gwv1.BackendRef) *string {
	// ONLY supports service backend now
	if *ref.Kind == constants.KubernetesServiceKind && *ref.Group == constants.KubernetesCoreGroup {
		port := int32(443)
		if ref.Port != nil {
			port = int32(*ref.Port)
		}

		target := fmt.Sprintf("%s:%d", ref.Name, port)

		return &target
	}

	return nil
}

func backendWeight(bk gwv1.BackendRef) int32 {
	if bk.Weight != nil {
		return *bk.Weight
	}

	return 1
}

func mergeL7RouteRule(rule1 fgw.L7RouteRule, rule2 fgw.L7RouteRule) fgw.L7RouteRule {
	mergedRule := fgw.L7RouteRule{}

	for hostname, rule := range rule1 {
		mergedRule[hostname] = rule
	}

	for hostname, rule := range rule2 {
		if r1, exists := mergedRule[hostname]; exists {
			// can only merge same type of route into one hostname
			switch r1 := r1.(type) {
			case *fgw.GRPCRouteRuleSpec:
				switch r2 := rule.(type) {
				case *fgw.GRPCRouteRuleSpec:
					if !reflect.DeepEqual(r1.RateLimit, r2.RateLimit) {
						continue
					}
					if !reflect.DeepEqual(r1.AccessControlLists, r2.AccessControlLists) {
						continue
					}
					if !reflect.DeepEqual(r1.FaultInjection, r2.FaultInjection) {
						continue
					}

					r1.Matches = mergeGRPCTrafficMatches(r1.Matches, r2.Matches)
					r1.Sort()

					mergedRule[hostname] = r1
				default:
					log.Error().Msgf("%s has been already mapped to RouteRule[%s] %v, current RouteRule %v will be dropped.", hostname, r1.RouteType, r1, r2)
				}
			case *fgw.HTTPRouteRuleSpec:
				switch r2 := rule.(type) {
				case *fgw.HTTPRouteRuleSpec:
					if !reflect.DeepEqual(r1.RateLimit, r2.RateLimit) {
						continue
					}
					if !reflect.DeepEqual(r1.AccessControlLists, r2.AccessControlLists) {
						continue
					}
					if !reflect.DeepEqual(r1.FaultInjection, r2.FaultInjection) {
						continue
					}

					r1.Matches = mergeHTTPTrafficMatches(r1.Matches, r2.Matches)
					r1.Sort()

					mergedRule[hostname] = r1
				default:
					log.Error().Msgf("%s has been already mapped to RouteRule[%s] %v, current RouteRule %v will be dropped.", hostname, r1.RouteType, r1, r2)
				}
			}
		} else {
			mergedRule[hostname] = rule
		}
	}

	return mergedRule
}

func mergeHTTPTrafficMatches(matches1 []fgw.HTTPTrafficMatch, matches2 []fgw.HTTPTrafficMatch) []fgw.HTTPTrafficMatch {
	hashmap := make(map[string]fgw.HTTPTrafficMatch)

	for _, m1 := range matches1 {
		hashmap[httpTrafficMatchHash(m1)] = m1
	}

	for _, m2 := range matches2 {
		h := httpTrafficMatchHash(m2)

		if m1, exists := hashmap[h]; exists {
			m1.BackendService = mergeBackendService(m1.BackendService, m2.BackendService)
			hashmap[h] = m1
			continue
		}

		hashmap[h] = m2
	}

	mergedMatches := make([]fgw.HTTPTrafficMatch, 0)
	for _, m := range hashmap {
		mergedMatches = append(mergedMatches, m)
	}

	return mergedMatches
}

func httpTrafficMatchHash(m fgw.HTTPTrafficMatch) string {
	return utils.SimpleHash(&fgw.HTTPTrafficMatch{
		Path:               m.Path,
		Headers:            m.Headers,
		Methods:            m.Methods,
		RequestParams:      m.RequestParams,
		RateLimit:          m.RateLimit,
		AccessControlLists: m.AccessControlLists,
		FaultInjection:     m.FaultInjection,
		Filters:            m.Filters,
	})
}

func mergeGRPCTrafficMatches(matches1 []fgw.GRPCTrafficMatch, matches2 []fgw.GRPCTrafficMatch) []fgw.GRPCTrafficMatch {
	hashmap := make(map[string]fgw.GRPCTrafficMatch)

	for _, m1 := range matches1 {
		hashmap[grpcTrafficMatchHash(m1)] = m1
	}

	for _, m2 := range matches2 {
		h := grpcTrafficMatchHash(m2)

		if m1, exists := hashmap[h]; exists {
			m1.BackendService = mergeBackendService(m1.BackendService, m2.BackendService)
			hashmap[h] = m1
			continue
		}

		hashmap[h] = m2
	}

	mergedMatches := make([]fgw.GRPCTrafficMatch, 0)
	for _, m := range hashmap {
		mergedMatches = append(mergedMatches, m)
	}

	return mergedMatches
}

func grpcTrafficMatchHash(m fgw.GRPCTrafficMatch) string {
	return utils.SimpleHash(&fgw.GRPCTrafficMatch{
		Headers:            m.Headers,
		Method:             m.Method,
		RateLimit:          m.RateLimit,
		AccessControlLists: m.AccessControlLists,
		FaultInjection:     m.FaultInjection,
		Filters:            m.Filters,
	})
}

func mergeBackendService(bs1 map[string]fgw.BackendServiceConfig, bs2 map[string]fgw.BackendServiceConfig) map[string]fgw.BackendServiceConfig {
	services := make(map[string]fgw.BackendServiceConfig)

	for k, c := range bs1 {
		services[k] = c
	}

	for k, c2 := range bs2 {
		if c1, exists := services[k]; exists {
			if !reflect.DeepEqual(c1, c2) {
				log.Error().Msgf("BackendService %s has been already mapped to %v, current %v will be dropped.", k, c1, c2)
				continue
			}
		}

		services[k] = c2
	}

	return services
}

//lint:ignore U1000 ignore unused
func copyMap[K, V comparable](m map[K]V) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func isEndpointReady(ep discoveryv1.Endpoint) bool {
	return ep.Conditions.Ready != nil && *ep.Conditions.Ready
}

func getServicePort(svc *corev1.Service, port *int32) (corev1.ServicePort, error) {
	if port == nil && len(svc.Spec.Ports) == 1 {
		return svc.Spec.Ports[0], nil
	}

	if port != nil {
		for _, p := range svc.Spec.Ports {
			if p.Port == *port {
				return p, nil
			}
		}
	}

	return corev1.ServicePort{}, fmt.Errorf("no matching port for Service %s and port %d", svc.Name, port)
}

func filterEndpointSliceList(endpointSliceList []*discoveryv1.EndpointSlice, port corev1.ServicePort) []*discoveryv1.EndpointSlice {
	filtered := make([]*discoveryv1.EndpointSlice, 0, len(endpointSliceList))

	for _, endpointSlice := range endpointSliceList {
		if !ignoreEndpointSlice(endpointSlice, port) {
			filtered = append(filtered, endpointSlice)
		}
	}

	return filtered
}

func ignoreEndpointSlice(endpointSlice *discoveryv1.EndpointSlice, port corev1.ServicePort) bool {
	if endpointSlice.AddressType != discoveryv1.AddressTypeIPv4 {
		return true
	}

	// ignore endpoint slices that don't have a matching port.
	return findEndpointSlicePort(endpointSlice.Ports, port) == 0
}

func findEndpointSlicePort(ports []discoveryv1.EndpointPort, svcPort corev1.ServicePort) int32 {
	portName := svcPort.Name
	for _, p := range ports {
		if p.Port == nil {
			return getDefaultPort(svcPort)
		}

		if p.Name != nil && *p.Name == portName {
			return *p.Port
		}
	}

	return 0
}

func findEndpointPort(ports []corev1.EndpointPort, svcPort corev1.ServicePort) int32 {
	for i, epPort := range ports {
		if svcPort.Name == "" {
			// port.Name is optional if there is only one port
			return epPort.Port
		}

		if svcPort.Name == epPort.Name {
			return epPort.Port
		}

		if i == len(ports)-1 && svcPort.TargetPort.Type == intstr.Int {
			return svcPort.TargetPort.IntVal
		}
	}

	return 0
}

func getDefaultPort(svcPort corev1.ServicePort) int32 {
	switch svcPort.TargetPort.Type {
	case intstr.Int:
		if svcPort.TargetPort.IntVal != 0 {
			return svcPort.TargetPort.IntVal
		}
	}

	return svcPort.Port
}

func toFSMHTTPRouteFilter(filter gwv1.HTTPRouteFilter, defaultNs string, services map[string]serviceInfo) fgw.Filter {
	result := fgw.HTTPRouteFilter{Type: filter.Type}

	if filter.RequestHeaderModifier != nil {
		result.RequestHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Add),
			Remove: filter.RequestHeaderModifier.Remove,
		}
	}

	if filter.ResponseHeaderModifier != nil {
		result.ResponseHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Add),
			Remove: filter.ResponseHeaderModifier.Remove,
		}
	}

	if filter.RequestRedirect != nil {
		result.RequestRedirect = &fgw.HTTPRequestRedirectFilter{
			Scheme:     filter.RequestRedirect.Scheme,
			Hostname:   toFSMHostname(filter.RequestRedirect.Hostname),
			Path:       toFSMHTTPPathModifier(filter.RequestRedirect.Path),
			Port:       toFSMPortNumber(filter.RequestRedirect.Port),
			StatusCode: filter.RequestRedirect.StatusCode,
		}
	}

	if filter.URLRewrite != nil {
		result.URLRewrite = &fgw.HTTPURLRewriteFilter{
			Hostname: toFSMHostname(filter.URLRewrite.Hostname),
			Path:     toFSMHTTPPathModifier(filter.URLRewrite.Path),
		}
	}

	if filter.RequestMirror != nil {
		if svcPort := backendRefToServicePortName(filter.RequestMirror.BackendRef, defaultNs); svcPort != nil {
			result.RequestMirror = &fgw.HTTPRequestMirrorFilter{
				BackendService: svcPort.String(),
			}

			services[svcPort.String()] = serviceInfo{
				svcPortName: *svcPort,
			}
		}
	}

	// TODO: implement it later
	if filter.ExtensionRef != nil {
		result.ExtensionRef = filter.ExtensionRef
	}

	return result
}

func toFSMGRPCRouteFilter(filter gwv1alpha2.GRPCRouteFilter, defaultNs string, services map[string]serviceInfo) fgw.Filter {
	result := fgw.GRPCRouteFilter{Type: filter.Type}

	if filter.RequestHeaderModifier != nil {
		result.RequestHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Add),
			Remove: filter.RequestHeaderModifier.Remove,
		}
	}

	if filter.ResponseHeaderModifier != nil {
		result.ResponseHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Add),
			Remove: filter.ResponseHeaderModifier.Remove,
		}
	}

	if filter.RequestMirror != nil {
		if svcPort := backendRefToServicePortName(filter.RequestMirror.BackendRef, defaultNs); svcPort != nil {
			result.RequestMirror = &fgw.HTTPRequestMirrorFilter{
				BackendService: svcPort.String(),
			}

			services[svcPort.String()] = serviceInfo{
				svcPortName: *svcPort,
			}
		}
	}

	// TODO: implement it later
	if filter.ExtensionRef != nil {
		result.ExtensionRef = filter.ExtensionRef
	}

	return result
}

func toFSMHTTPPathModifier(path *gwv1.HTTPPathModifier) *fgw.HTTPPathModifier {
	if path == nil {
		return nil
	}

	return &fgw.HTTPPathModifier{
		Type:               path.Type,
		ReplaceFullPath:    path.ReplaceFullPath,
		ReplacePrefixMatch: path.ReplacePrefixMatch,
	}
}

func toFSMHostname(hostname *gwv1.PreciseHostname) *string {
	if hostname == nil {
		return nil
	}

	return pointer.String(string(*hostname))
}

func toFSMHTTPHeaders(headers []gwv1.HTTPHeader) []fgw.HTTPHeader {
	if len(headers) == 0 {
		return nil
	}

	results := make([]fgw.HTTPHeader, 0)
	for _, h := range headers {
		results = append(results, fgw.HTTPHeader{
			Name:  string(h.Name),
			Value: h.Value,
		})
	}

	return results
}

func toFSMPortNumber(port *gwv1.PortNumber) *int32 {
	if port == nil {
		return nil
	}

	return pointer.Int32(int32(*port))
}

func insertAgentServiceScript(chains []string) []string {
	httpCodecIndex := slices.Index(chains, httpCodecScript)
	if httpCodecIndex != -1 {
		return slices.Insert(chains, httpCodecIndex+1, agentServiceScript)
	}

	return chains
}

func insertProxyTagScript(chains []string) []string {
	httpCodecIndex := slices.Index(chains, httpCodecScript)
	if httpCodecIndex != -1 {
		return slices.Insert(chains, httpCodecIndex+1, proxyTagScript)
	}

	return chains
}

func setGroupVersionKind[T GatewayAPIResource](objects []T, gvk schema.GroupVersionKind) []client.Object {
	resources := make([]client.Object, 0)

	for _, obj := range objects {
		obj := client.Object(obj)
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		resources = append(resources, obj)
	}

	return resources
}
