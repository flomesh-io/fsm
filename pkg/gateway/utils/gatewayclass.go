package utils

import (
	"context"
	"fmt"

	metautil "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/apis/gateway"
	"github.com/flomesh-io/fsm/pkg/constants"
)

// IsAcceptedGatewayClass returns true if the gateway class is accepted
func IsAcceptedGatewayClass(gatewayClass *gwv1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gwv1.GatewayClassConditionStatusAccepted))
}

// IsActiveGatewayClass returns true if the gateway class is active
func IsActiveGatewayClass(gatewayClass *gwv1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gateway.GatewayClassConditionStatusActive))
}

// IsEffectiveGatewayClass returns true if the gateway class is effective
func IsEffectiveGatewayClass(gatewayClass *gwv1.GatewayClass) bool {
	return IsAcceptedGatewayClass(gatewayClass) && IsActiveGatewayClass(gatewayClass)
}

// FindEffectiveGatewayClass returns the effective GatewayClass
func FindEffectiveGatewayClass(c cache.Cache) (*gwv1.GatewayClass, error) {
	list := &gwv1.GatewayClassList{}
	if err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ControllerGatewayClassIndex, constants.GatewayController),
	}); err != nil {
		return nil, fmt.Errorf("failed to list gateway classes: %s", err)
	}

	for _, cls := range SortResources(ToSlicePtr(list.Items)) {
		cls := cls
		if IsEffectiveGatewayClass(cls) {
			return cls, nil
		}
	}

	return nil, nil
}
