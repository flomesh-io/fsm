package ktog

import (
	"testing"

	"github.com/flomesh-io/fsm/pkg/connector"
)

func TestRouteNameForService(t *testing.T) {
	tests := []struct {
		name        string
		routeName   string
		serviceName string
		want        bool
	}{
		{name: "legacy exact name", routeName: "foo", serviceName: "foo", want: true},
		{name: "http route", routeName: "foo-8080-http", serviceName: "foo", want: true},
		{name: "grpc route", routeName: "foo-9090-grpc", serviceName: "foo", want: true},
		{name: "tcp route", routeName: "foo-3306-tcp", serviceName: "foo", want: true},
		{name: "service name prefix", routeName: "foo-bar-8080-http", serviceName: "foo", want: false},
		{name: "port-like service suffix", routeName: "foo-8080-http", serviceName: "foo-8080", want: false},
		{name: "non-numeric port", routeName: "foo-notaport-http", serviceName: "foo", want: false},
		{name: "zero port", routeName: "foo-0-http", serviceName: "foo", want: false},
		{name: "port too large", routeName: "foo-65536-http", serviceName: "foo", want: false},
		{name: "unknown protocol", routeName: "foo-8080-udp", serviceName: "foo", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := routeNameForService(tc.routeName, tc.serviceName); got != tc.want {
				t.Fatalf("routeNameForService(%q, %q) = %t, want %t", tc.routeName, tc.serviceName, got, tc.want)
			}
		})
	}
}

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
