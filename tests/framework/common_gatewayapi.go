package framework

import (
	"context"
	"fmt"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// CreateGateway Creates a Gateway
func (td *FsmTestData) CreateGateway(ns string, gw gwv1.Gateway) (*gwv1.Gateway, error) {
	ret, err := td.GatewayAPIClient.GatewayV1().Gateways(ns).Create(context.Background(), &gw, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gateway: %w", err)
	}

	return ret, nil
}

// CreateGatewayAPIHTTPRoute Creates a HTTPRoute
func (td *FsmTestData) CreateGatewayAPIHTTPRoute(ns string, r gwv1.HTTPRoute) (*gwv1.HTTPRoute, error) {
	hr, err := td.GatewayAPIClient.GatewayV1().HTTPRoutes(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTPRoute: %w", err)
	}

	return hr, nil
}

// CreateGatewayAPIReferenceGrant Creates a ReferenceGrant
func (td *FsmTestData) CreateGatewayAPIReferenceGrant(ns string, r gwv1beta1.ReferenceGrant) (*gwv1beta1.ReferenceGrant, error) {
	rg, err := td.GatewayAPIClient.GatewayV1beta1().ReferenceGrants(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create ReferenceGrant: %w", err)
	}

	return rg, nil
}

// CreateGatewayAPIGRPCRoute Creates a GRPCRoute
func (td *FsmTestData) CreateGatewayAPIGRPCRoute(ns string, r gwv1.GRPCRoute) (*gwv1.GRPCRoute, error) {
	hr, err := td.GatewayAPIClient.GatewayV1().GRPCRoutes(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create GRPCRoute: %w", err)
	}

	return hr, nil
}

// CreateGatewayAPITLSRoute Creates a TLSRoute
func (td *FsmTestData) CreateGatewayAPITLSRoute(ns string, r gwv1alpha2.TLSRoute) (*gwv1alpha2.TLSRoute, error) {
	hr, err := td.GatewayAPIClient.GatewayV1alpha2().TLSRoutes(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create TLSRoute: %w", err)
	}

	return hr, nil
}

// CreateGatewayAPITCPRoute Creates a TCPRoute
func (td *FsmTestData) CreateGatewayAPITCPRoute(ns string, r gwv1alpha2.TCPRoute) (*gwv1alpha2.TCPRoute, error) {
	hr, err := td.GatewayAPIClient.GatewayV1alpha2().TCPRoutes(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create TCPRoute: %w", err)
	}

	return hr, nil
}

// CreateGatewayAPIUDPRoute Creates a UDPRoute
func (td *FsmTestData) CreateGatewayAPIUDPRoute(ns string, r gwv1alpha2.UDPRoute) (*gwv1alpha2.UDPRoute, error) {
	hr, err := td.GatewayAPIClient.GatewayV1alpha2().UDPRoutes(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create UDPRoute: %w", err)
	}

	return hr, nil
}

// CreateGatewayAPIDNSModifier Creates a DNSModifier config
func (td *FsmTestData) CreateGatewayAPIDNSModifier(ns string, r extv1alpha1.DNSModifier) (*extv1alpha1.DNSModifier, error) {
	dm, err := td.ExtensionClient.ExtensionV1alpha1().DNSModifiers(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create DNSModifier: %w", err)
	}

	return dm, nil
}

// CreateGatewayAPIFilter Creates a Filter
func (td *FsmTestData) CreateGatewayAPIFilter(ns string, r extv1alpha1.Filter) (*extv1alpha1.Filter, error) {
	f, err := td.ExtensionClient.ExtensionV1alpha1().Filters(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create Filter: %w", err)
	}

	return f, nil
}

// CreateGatewayAPIRouteRuleFilterPolicy Creates a RouteRuleFilterPolicy
func (td *FsmTestData) CreateGatewayAPIRouteRuleFilterPolicy(ns string, r gwpav1alpha2.RouteRuleFilterPolicy) (*gwpav1alpha2.RouteRuleFilterPolicy, error) {
	p, err := td.PolicyAttachmentClient.GatewayV1alpha2().RouteRuleFilterPolicies(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create RouteRuleFilterPolicy: %w", err)
	}

	return p, nil
}
