package framework

import (
	"context"
	"fmt"

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

// CreateHTTPRoute Creates a HTTPRoute
func (td *FsmTestData) CreateHTTPRoute(ns string, r gwv1.HTTPRoute) (*gwv1.HTTPRoute, error) {
	hr, err := td.GatewayAPIClient.GatewayV1().HTTPRoutes(ns).Create(context.Background(), &r, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTPRoute: %w", err)
	}

	return hr, nil
}
