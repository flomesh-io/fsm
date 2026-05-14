package connector

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	nacos "github.com/nacos-group/nacos-sdk-go/v2/model"
)

func TestAgentServiceFromNacosProtocol(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		want     MicroServiceProtocol
	}{
		// appprotocol 字段测试
		{"appprotocol grpc", map[string]string{"appprotocol": "grpc"}, ProtocolGRPC},
		{"appprotocol tri", map[string]string{"appprotocol": "tri"}, MicroServiceProtocol("tri")},
		{"appprotocol http", map[string]string{"appprotocol": "http"}, ProtocolHTTP},
		// protocol 字段测试（兼容旧数据）
		{"protocol grpc lowercase", map[string]string{"protocol": "grpc"}, ProtocolGRPC},
		{"protocol GRPC uppercase", map[string]string{"protocol": "GRPC"}, ProtocolGRPC},
		{"protocol GrPc mixed", map[string]string{"protocol": "GrPc"}, ProtocolGRPC},
		{"protocol tri lowercase", map[string]string{"protocol": "tri"}, MicroServiceProtocol("tri")},
		{"protocol TRI uppercase", map[string]string{"protocol": "TRI"}, MicroServiceProtocol("tri")},
		{"protocol Tri mixed", map[string]string{"protocol": "Tri"}, MicroServiceProtocol("tri")},
		{"protocol http explicit", map[string]string{"protocol": "http"}, ProtocolHTTP},
		{"protocol empty value", map[string]string{"protocol": ""}, ProtocolHTTP},
		// appprotocol 优先级高于 protocol
		{"appprotocol takes precedence", map[string]string{"appprotocol": "grpc", "protocol": "http"}, ProtocolGRPC},
		// 无字段
		{"no key", map[string]string{"other": "x"}, ProtocolHTTP},
		{"nil metadata", nil, ProtocolHTTP},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ins := &nacos.Instance{
				InstanceId:  "id-1",
				ServiceName: "DEFAULT_GROUP" + constant.SERVICE_INFO_SPLITER + "svc-a",
				Ip:          "10.0.0.1",
				Port:        8080,
				Metadata:    tc.metadata,
			}
			as := &AgentService{}
			as.FromNacos(ins)
			if got := *as.MicroService.Protocol(); got != tc.want {
				t.Fatalf("protocol=%q want %q", got, tc.want)
			}
			if as.MicroService.Service != "svc-a" {
				t.Fatalf("service=%q want svc-a", as.MicroService.Service)
			}
		})
	}
}

func TestCatalogRegistrationToNacosProtocol(t *testing.T) {
	tests := []struct {
		name     string
		proto    MicroServiceProtocol
		wantKey  bool
		wantVal  string
	}{
		{"grpc writes appprotocol=grpc", ProtocolGRPC, true, "grpc"},
		{"http omits appprotocol key", ProtocolHTTP, false, ""},
		{"empty omits appprotocol key", MicroServiceProtocol(""), false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &AgentService{}
			svc.MicroService.Service = "svc-a"
			svc.MicroService.Protocol().SetVar(tc.proto)
			svc.MicroService.Endpoint().Set(MicroServiceAddress("10.0.0.1"), MicroServicePort(8080))
			cr := &CatalogRegistration{Service: svc}
			out := cr.ToNacos("DEFAULT", "DEFAULT_GROUP", 1.0)
			if out == nil {
				t.Fatal("ToNacos returned nil")
			}
			got, ok := out.Metadata["appprotocol"]
			if tc.wantKey {
				if !ok {
					t.Fatalf("metadata[appprotocol] missing, want %q", tc.wantVal)
				}
				if got != tc.wantVal {
					t.Fatalf("metadata[appprotocol]=%q want %q", got, tc.wantVal)
				}
			} else if ok && got == "grpc" {
				t.Fatalf("metadata[appprotocol] unexpectedly set to grpc for non-grpc service")
			}
		})
	}
}

func TestCatalogRegistrationToNacosPreservesMeta(t *testing.T) {
	svc := &AgentService{
		Meta: map[string]interface{}{"env": "prod", "region": "us-west"},
	}
	svc.MicroService.Service = "svc-a"
	svc.MicroService.Protocol().SetVar(ProtocolGRPC)
	svc.MicroService.Endpoint().Set(MicroServiceAddress("10.0.0.1"), MicroServicePort(8080))
	cr := &CatalogRegistration{Service: svc}
	out := cr.ToNacos("DEFAULT", "DEFAULT_GROUP", 1.0)
	if out.Metadata["env"] != "prod" || out.Metadata["region"] != "us-west" {
		t.Fatalf("existing metadata not preserved: %#v", out.Metadata)
	}
	if out.Metadata["appprotocol"] != "grpc" {
		t.Fatalf("appprotocol metadata not written: %#v", out.Metadata)
	}
}
