package provider

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

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

type nacosTestController struct {
	connector.ConnectController
	ttl time.Duration
}

func (c *nacosTestController) GetAuthNacosTokenTtl() time.Duration { return c.ttl }
func (c *nacosTestController) GetAuthNacosNamespaceId() string     { return "" }
func (c *nacosTestController) GetAuthNacosUsername() string        { return "" }
func (c *nacosTestController) GetAuthNacosPassword() string        { return "" }
func (c *nacosTestController) GetAuthNacosAccessKey() string       { return "" }
func (c *nacosTestController) GetAuthNacosSecretKey() string       { return "" }
func (c *nacosTestController) GetHTTPAddr() string {
	return "http://127.0.0.1:8848/nacos?grpcport=9848"
}
func (c *nacosTestController) WaitLimiter() {}
func (c *nacosTestController) SetServiceInstanceIDFunc(connector.ServiceInstanceIDFunc) {
}

type stubNamingClient struct {
	naming_client.INamingClient
	closeCount       atomic.Int32
	subscribeCount   atomic.Int32
	unsubscribeCount atomic.Int32
	failSubscribeAt  int32
}

func (c *stubNamingClient) CloseClient() {
	c.closeCount.Add(1)
}

func (c *stubNamingClient) Subscribe(*vo.SubscribeParam) error {
	call := c.subscribeCount.Add(1)
	if call == c.failSubscribeAt {
		return errors.New("subscribe failed")
	}
	return nil
}

func (c *stubNamingClient) Unsubscribe(*vo.SubscribeParam) error {
	c.unsubscribeCount.Add(1)
	return nil
}

func newNacosTestClient(t *testing.T) *NacosDiscoveryClient {
	t.Helper()
	client, err := GetNacosDiscoveryClient(&nacosTestController{ttl: time.Hour})
	if err != nil {
		t.Fatalf("GetNacosDiscoveryClient() error = %v", err)
	}
	client.clientReadyDelay = 0
	return client
}

func TestNacosClientInitializesSameKeyOnce(t *testing.T) {
	client := newNacosTestClient(t)
	var createCount atomic.Int32
	client.createNamingClient = func(map[string]interface{}) (naming_client.INamingClient, error) {
		createCount.Add(1)
		time.Sleep(20 * time.Millisecond)
		return &stubNamingClient{}, nil
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := client.nacosClient("same-key"); err != nil {
				t.Errorf("nacosClient() error = %v", err)
			}
		}()
	}
	wg.Wait()

	if got := createCount.Load(); got != 1 {
		t.Fatalf("create count = %d, want 1", got)
	}
}

func TestNacosClientInitializesDifferentKeysConcurrently(t *testing.T) {
	client := newNacosTestClient(t)
	client.createNamingClient = func(map[string]interface{}) (naming_client.INamingClient, error) {
		time.Sleep(100 * time.Millisecond)
		return &stubNamingClient{}, nil
	}

	start := time.Now()
	var wg sync.WaitGroup
	for _, key := range []string{"key-a", "key-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			if _, _, err := client.nacosClient(key); err != nil {
				t.Errorf("nacosClient(%q) error = %v", key, err)
			}
		}(key)
	}
	wg.Wait()

	if elapsed := time.Since(start); elapsed >= 180*time.Millisecond {
		t.Fatalf("different keys initialized serially, elapsed %s", elapsed)
	}
}

func TestNacosClientUsesStdoutOnlyConfigAndIncrementsGeneration(t *testing.T) {
	client := newNacosTestClient(t)
	var configs []constant.ClientConfig
	client.createNamingClient = func(properties map[string]interface{}) (naming_client.INamingClient, error) {
		configs = append(configs, properties["clientConfig"].(constant.ClientConfig))
		return &stubNamingClient{}, nil
	}

	_, firstGeneration, err := client.nacosClient(aloneConnect)
	if err != nil {
		t.Fatalf("first nacosClient() error = %v", err)
	}
	conn := client.nacosConnects[aloneConnect]
	conn.lock.Lock()
	conn.expiresAt = time.Now().Add(-time.Second)
	conn.lock.Unlock()
	_, secondGeneration, err := client.nacosClient(aloneConnect)
	if err != nil {
		t.Fatalf("second nacosClient() error = %v", err)
	}

	if firstGeneration != 1 || secondGeneration != 2 {
		t.Fatalf("generations = %d, %d; want 1, 2", firstGeneration, secondGeneration)
	}
	for _, config := range configs {
		if !config.AppendToStdout {
			t.Fatal("AppendToStdout = false")
		}
		if config.LogDir != "" {
			t.Fatalf("LogDir = %q, want empty with custom stdout logger", config.LogDir)
		}
	}
}

func TestSubscribeToServiceRollsBackPartialSubscription(t *testing.T) {
	client := newNacosTestClient(t)
	namingClient := &stubNamingClient{failSubscribeAt: 2}
	client.createNamingClient = func(map[string]interface{}) (naming_client.INamingClient, error) {
		return namingClient, nil
	}

	unsubscribe, _, err := client.SubscribeToService("demo", []string{"group-a", "group-b"}, nil, func(interface{}, error) {})
	if err == nil {
		t.Fatal("expected subscribe error")
	}
	if unsubscribe != nil {
		t.Fatal("expected nil unsubscribe after rollback")
	}
	if got := namingClient.unsubscribeCount.Load(); got != 1 {
		t.Fatalf("unsubscribe count = %d, want 1", got)
	}
}
