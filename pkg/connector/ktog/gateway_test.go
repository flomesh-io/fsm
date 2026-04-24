package ktog

import (
	"testing"

	"github.com/flomesh-io/fsm/pkg/connector"
)

func TestBuildGRPCRouteMatches(t *testing.T) {
	t.Run("nil svcMeta returns nil (catch-all)", func(t *testing.T) {
		if got := buildGRPCRouteMatches(nil); got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("nil GRPCMeta returns nil (catch-all)", func(t *testing.T) {
		if got := buildGRPCRouteMatches(&connector.MicroSvcMeta{}); got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("empty Interface returns nil (catch-all)", func(t *testing.T) {
		svcMeta := &connector.MicroSvcMeta{
			GRPCMeta: &connector.GRPCMeta{
				Interface: "",
				Methods:   map[string][]string{"SayHello": nil},
			},
		}
		if got := buildGRPCRouteMatches(svcMeta); got != nil {
			t.Fatalf("expected nil for empty interface, got %+v", got)
		}
	})

	t.Run("empty Methods returns nil (catch-all)", func(t *testing.T) {
		svcMeta := &connector.MicroSvcMeta{
			GRPCMeta: &connector.GRPCMeta{
				Interface: "Greeter",
				Methods:   map[string][]string{},
			},
		}
		if got := buildGRPCRouteMatches(svcMeta); got != nil {
			t.Fatalf("expected nil for empty methods, got %+v", got)
		}
	})

	t.Run("populated GRPCMeta produces method-specific matches", func(t *testing.T) {
		svcMeta := &connector.MicroSvcMeta{
			GRPCMeta: &connector.GRPCMeta{
				Interface: "Greeter",
				Methods: map[string][]string{
					"SayHello": nil,
					"SayBye":   nil,
				},
			},
		}
		matches := buildGRPCRouteMatches(svcMeta)
		if len(matches) != 2 {
			t.Fatalf("expected 2 matches, got %d: %+v", len(matches), matches)
		}
		seenMethods := map[string]bool{}
		for _, m := range matches {
			if m.Method == nil {
				t.Fatalf("match.Method is nil: %+v", m)
			}
			if m.Method.Service == nil || *m.Method.Service != "Greeter" {
				t.Fatalf("match.Method.Service = %v, want Greeter", m.Method.Service)
			}
			if m.Method.Method == nil {
				t.Fatalf("match.Method.Method is nil")
			}
			seenMethods[*m.Method.Method] = true
		}
		if !seenMethods["SayHello"] || !seenMethods["SayBye"] {
			t.Fatalf("missing methods, got %+v", seenMethods)
		}
	})
}
