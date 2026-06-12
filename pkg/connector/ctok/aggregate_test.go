package ctok

import (
	"reflect"
	"testing"

	"github.com/flomesh-io/fsm/pkg/connector"
)

func TestAggregateMetaMergesGRPCMetadataForExistingEndpoint(t *testing.T) {
	httpInstance := newAggregateTestInstance("10.0.0.1", 8080, connector.ProtocolHTTP)
	grpcInstance := newAggregateTestInstance("10.0.0.1", 9090, connector.ProtocolGRPC)
	grpcInstance.GRPCInterface = "Greeter"
	grpcInstance.GRPCMethods = []string{"SayHello"}
	grpcInstance.Meta = map[string]interface{}{"version": "v1"}

	for _, instances := range [][]*connector.AgentService{
		{httpInstance, grpcInstance},
		{grpcInstance, httpInstance},
	} {
		svcMetaMap := make(map[connector.KubeSvcName]*connector.MicroSvcMeta)
		source := &CtoKSource{}
		for _, instance := range instances {
			source.aggregateMeta(svcMetaMap, "demo", instance)
		}

		meta := svcMetaMap["demo"]
		if meta.GRPCMeta == nil {
			t.Fatal("expected gRPC metadata")
		}
		if meta.GRPCMeta.Interface != "Greeter" {
			t.Fatalf("interface = %q, want Greeter", meta.GRPCMeta.Interface)
		}
		if got := meta.GRPCMeta.Methods["SayHello"]; !reflect.DeepEqual(got, []string{"10.0.0.1"}) {
			t.Fatalf("method endpoints = %#v, want [10.0.0.1]", got)
		}
		endpoint := meta.Endpoints["10.0.0.1"]
		if len(endpoint.Ports) != 2 {
			t.Fatalf("ports = %#v, want two ports", endpoint.Ports)
		}
		if endpoint.GRPCMeta["version"] != "v1" {
			t.Fatalf("gRPC endpoint metadata = %#v", endpoint.GRPCMeta)
		}
	}
}

func TestAggregateMetaDeduplicatesGRPCEndpoints(t *testing.T) {
	instance := newAggregateTestInstance("10.0.0.1", 9090, connector.ProtocolGRPC)
	instance.GRPCInterface = "Greeter"
	instance.GRPCMethods = []string{"SayHello", "SayHello"}

	svcMetaMap := make(map[connector.KubeSvcName]*connector.MicroSvcMeta)
	source := &CtoKSource{}
	source.aggregateMeta(svcMetaMap, "demo", instance)

	got := svcMetaMap["demo"].GRPCMeta.Methods["SayHello"]
	if !reflect.DeepEqual(got, []string{"10.0.0.1"}) {
		t.Fatalf("method endpoints = %#v, want one endpoint", got)
	}
}

func TestAggregateMetaChoosesGRPCInterfaceDeterministically(t *testing.T) {
	alpha := newAggregateTestInstance("10.0.0.1", 9090, connector.ProtocolGRPC)
	alpha.GRPCInterface = "Alpha"
	alpha.GRPCMethods = []string{"AlphaMethod"}
	beta := newAggregateTestInstance("10.0.0.2", 9090, connector.ProtocolGRPC)
	beta.GRPCInterface = "Beta"
	beta.GRPCMethods = []string{"BetaMethod"}

	for _, instances := range [][]*connector.AgentService{
		{alpha, beta},
		{beta, alpha},
	} {
		svcMetaMap := make(map[connector.KubeSvcName]*connector.MicroSvcMeta)
		source := &CtoKSource{}
		for _, instance := range instances {
			source.aggregateMeta(svcMetaMap, "demo", instance)
		}

		grpcMeta := svcMetaMap["demo"].GRPCMeta
		if grpcMeta.Interface != "Alpha" {
			t.Fatalf("interface = %q, want Alpha", grpcMeta.Interface)
		}
		if _, exists := grpcMeta.Methods["BetaMethod"]; exists {
			t.Fatalf("methods include conflicting interface: %#v", grpcMeta.Methods)
		}
		if got := grpcMeta.Methods["AlphaMethod"]; !reflect.DeepEqual(got, []string{"10.0.0.1"}) {
			t.Fatalf("AlphaMethod endpoints = %#v", got)
		}
	}
}

func newAggregateTestInstance(address string, port connector.MicroServicePort, protocol connector.MicroServiceProtocol) *connector.AgentService {
	instance := &connector.AgentService{Meta: make(map[string]interface{})}
	instance.MicroService.Endpoint().Set(connector.MicroServiceAddress(address), port)
	instance.MicroService.Protocol().SetVar(protocol)
	return instance
}
