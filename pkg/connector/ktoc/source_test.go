package ktoc

import (
	"testing"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
)

func TestDeregistrationFromRegistration(t *testing.T) {
	registration := &connector.CatalogRegistration{
		Node: "node-1",
		Service: &connector.AgentService{
			ID:         "instance-1",
			InstanceId: "service-ref-1",
			MicroService: connector.MicroService{
				NamespacedService: ctv1.NamespacedService{
					Namespace: "default",
					Service:   "demo",
				},
			},
		},
	}

	got := deregistrationFromRegistration(registration)
	if got == nil {
		t.Fatal("expected deregistration")
	}
	if got.ServiceID != "instance-1" || got.ServiceRef != "service-ref-1" || got.Node != "node-1" {
		t.Fatalf("unexpected deregistration: %+v", got)
	}
	if got.Namespace != "default" || got.Service != "demo" {
		t.Fatalf("unexpected namespaced service: %+v", got.NamespacedService)
	}
}

func TestDeregistrationFromRegistrationRejectsIncompleteInput(t *testing.T) {
	if got := deregistrationFromRegistration(nil); got != nil {
		t.Fatalf("nil registration returned %+v", got)
	}
	if got := deregistrationFromRegistration(&connector.CatalogRegistration{}); got != nil {
		t.Fatalf("registration without service returned %+v", got)
	}
}

func TestDeregistrationFromRegistrationFallsBackToServiceID(t *testing.T) {
	registration := &connector.CatalogRegistration{
		Service: &connector.AgentService{ID: "instance-1"},
	}
	got := deregistrationFromRegistration(registration)
	if got.ServiceRef != "instance-1" {
		t.Fatalf("ServiceRef = %q, want instance-1", got.ServiceRef)
	}
}
