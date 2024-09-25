package main

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/google/go-cmp/cmp/cmpopts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func main() {
	opts := cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime")

	a := &gwv1.GatewayStatus{
		Conditions: []metav1.Condition{
			{
				Type:               "Ready1",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: 1,
				Reason:             "Reason1",
				Message:            "Message1",
			},
		},

		Listeners: []gwv1.ListenerStatus{
			{
				Name: "listener1",
				SupportedKinds: []gwv1.RouteGroupKind{
					{
						Kind: "HTTPRoute",
					},
				},
				Conditions: []metav1.Condition{
					{
						Type:               "Ready2",
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Now(),
						ObservedGeneration: 1,
						Reason:             "Reason2",
						Message:            "Message2",
					},
				},
			},
		},
	}

	b := &gwv1.GatewayStatus{
		Conditions: []metav1.Condition{
			{
				Type:               "Ready1",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().Add(13 * time.Second)},
				ObservedGeneration: 1,
				Reason:             "Reason1",
				Message:            "Message1",
			},
		},

		Listeners: []gwv1.ListenerStatus{
			{
				Name: "listener1",
				SupportedKinds: []gwv1.RouteGroupKind{
					{
						Kind: "HTTPRoute",
					},
				},
				Conditions: []metav1.Condition{
					{
						Type:               "Ready3",
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{Time: time.Now().Add(17 * time.Second)},
						ObservedGeneration: 1,
						Reason:             "Reason2",
						Message:            "Message2",
					},
				},
			},
		},
	}

	fmt.Println(cmp.Diff(a, b, opts))
}
