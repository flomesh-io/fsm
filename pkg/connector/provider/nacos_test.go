package provider

import (
	"testing"

	"github.com/flomesh-io/fsm/pkg/connector"
)

func TestProtocolFromNacosMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		want     connector.MicroServiceProtocol
	}{
		{"grpc lowercase", map[string]string{"protocol": "grpc"}, connector.ProtocolGRPC},
		{"GRPC uppercase", map[string]string{"protocol": "GRPC"}, connector.ProtocolGRPC},
		{"GrPc mixed", map[string]string{"protocol": "GrPc"}, connector.ProtocolGRPC},
		{"http explicit", map[string]string{"protocol": "http"}, connector.ProtocolHTTP},
		{"empty value", map[string]string{"protocol": ""}, connector.ProtocolHTTP},
		{"unknown tri", map[string]string{"protocol": "tri"}, connector.ProtocolHTTP},
		{"no protocol key", map[string]string{"version": "1.0"}, connector.ProtocolHTTP},
		{"empty map", map[string]string{}, connector.ProtocolHTTP},
		{"nil map", nil, connector.ProtocolHTTP},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := protocolFromNacosMetadata(tc.metadata); got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
		})
	}
}
