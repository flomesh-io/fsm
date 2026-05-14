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
		// appprotocol 字段测试
		{"appprotocol grpc", map[string]string{"appprotocol": "grpc"}, connector.ProtocolGRPC},
		{"appprotocol tri", map[string]string{"appprotocol": "tri"}, connector.MicroServiceProtocol("tri")},
		{"appprotocol http", map[string]string{"appprotocol": "http"}, connector.ProtocolHTTP},
		// protocol 字段测试（兼容旧数据）
		{"protocol grpc lowercase", map[string]string{"protocol": "grpc"}, connector.ProtocolGRPC},
		{"protocol GRPC uppercase", map[string]string{"protocol": "GRPC"}, connector.ProtocolGRPC},
		{"protocol GrPc mixed", map[string]string{"protocol": "GrPc"}, connector.ProtocolGRPC},
		{"protocol tri lowercase", map[string]string{"protocol": "tri"}, connector.MicroServiceProtocol("tri")},
		{"protocol TRI uppercase", map[string]string{"protocol": "TRI"}, connector.MicroServiceProtocol("tri")},
		{"protocol Tri mixed", map[string]string{"protocol": "Tri"}, connector.MicroServiceProtocol("tri")},
		{"protocol http explicit", map[string]string{"protocol": "http"}, connector.ProtocolHTTP},
		{"protocol empty value", map[string]string{"protocol": ""}, connector.ProtocolHTTP},
		// appprotocol 优先级高于 protocol
		{"appprotocol takes precedence", map[string]string{"appprotocol": "grpc", "protocol": "http"}, connector.ProtocolGRPC},
		// 无字段
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
