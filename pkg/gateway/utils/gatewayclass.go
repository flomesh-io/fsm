package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/flomesh-io/fsm/pkg/constants"

	metautil "k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// IsAcceptedGatewayClass returns true if the gateway class is accepted
func IsAcceptedGatewayClass(gatewayClass *gwv1.GatewayClass) bool {
	return gatewayClass.Spec.ControllerName == constants.GatewayController &&
		metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gwv1.GatewayClassConditionStatusAccepted))
}

// FindGatewayClassByName returns the GatewayClass with the given name
func FindGatewayClassByName(c cache.Cache, name string) (*gwv1.GatewayClass, error) {
	gatewayClass := &gwv1.GatewayClass{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: name}, gatewayClass); err != nil {
		return nil, fmt.Errorf("failed to get gateway class %s: %s", name, err)
	}

	return gatewayClass, nil
}

// findFSMGatewayClasses returns the effective GatewayClasses for FSM
func findFSMGatewayClasses(c cache.Cache) ([]*gwv1.GatewayClass, error) {
	list := &gwv1.GatewayClassList{}
	if err := c.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ControllerGatewayClassIndex, constants.GatewayController),
	}); err != nil {
		return nil, fmt.Errorf("failed to list gateway classes: %s", err)
	}

	classes := make([]*gwv1.GatewayClass, 0)
	for _, cls := range SortResources(ToSlicePtr(list.Items)) {
		cls := cls
		if IsAcceptedGatewayClass(cls) {
			classes = append(classes, cls)
		}
	}

	return classes, nil
}
