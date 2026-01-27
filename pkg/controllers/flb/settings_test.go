package flb

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		expectedHTTP bool
	}{
		{
			name:         "http URL",
			baseURL:      "http://example.com:8080",
			expectedHTTP: true,
		},
		{
			name:         "https URL",
			baseURL:      "https://example.com:8443",
			expectedHTTP: false,
		},
		{
			name:         "HTTPS uppercase",
			baseURL:      "HTTPS://example.com",
			expectedHTTP: false,
		},
		{
			name:         "URL without scheme",
			baseURL:      "example.com",
			expectedHTTP: true,
		},
		{
			name:         "URL with path",
			baseURL:      "http://example.com/api/v1",
			expectedHTTP: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newHTTPClient(tt.baseURL)

			if client == nil {
				t.Fatal("expected non-nil client")
			}

			// Get the transport from resty client
			transport := client.GetClient().Transport
			if transport == nil {
				t.Fatal("expected non-nil transport")
			}

			httpTransport, ok := transport.(*http.Transport)
			if !ok {
				t.Fatal("expected *http.Transport")
			}

			if tt.expectedHTTP {
				// For HTTP URLs, TLSClientConfig should be nil
				if httpTransport.TLSClientConfig != nil {
					t.Errorf("expected nil TLSClientConfig for HTTP URL, got non-nil")
				}
			} else {
				// For HTTPS URLs, TLSClientConfig should have InsecureSkipVerify set
				if httpTransport.TLSClientConfig == nil {
					t.Errorf("expected non-nil TLSClientConfig for HTTPS URL")
				} else if !httpTransport.TLSClientConfig.InsecureSkipVerify {
					t.Errorf("expected InsecureSkipVerify to be true for HTTPS URL")
				}
			}
		})
	}
}

func TestNewHTTPClientTimeout(t *testing.T) {
	client := newHTTPClient("http://example.com")

	timeout := client.GetClient().Timeout
	expectedTimeout := 5 * time.Second
	if timeout != expectedTimeout {
		t.Errorf("expected timeout %v, got %v", expectedTimeout, timeout)
	}
}

func TestNewHTTPClientTransportSettings(t *testing.T) {
	client := newHTTPClient("http://example.com")

	transport := client.GetClient().Transport.(*http.Transport)

	if transport.DisableKeepAlives != false {
		t.Error("expected DisableKeepAlives to be false")
	}
	if transport.MaxIdleConns != 10 {
		t.Errorf("expected MaxIdleConns to be 10, got %d", transport.MaxIdleConns)
	}
	if transport.IdleConnTimeout != 60*time.Second {
		t.Errorf("expected IdleConnTimeout to be 60s, got %v", transport.IdleConnTimeout)
	}
	if transport.DisableCompression != false {
		t.Error("expected DisableCompression to be false")
	}
}
