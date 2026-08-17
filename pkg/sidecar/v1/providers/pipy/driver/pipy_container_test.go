package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testLocalDNSProxyUpstreamGetter struct {
	primary   string
	secondary string
}

func (g testLocalDNSProxyUpstreamGetter) GetLocalDNSProxyPrimaryUpstream() string {
	return g.primary
}

func (g testLocalDNSProxyUpstreamGetter) GetLocalDNSProxySecondaryUpstream() string {
	return g.secondary
}

func TestGetLocalDNSProxyNameservers(t *testing.T) {
	tests := []struct {
		name       string
		controller string
		primary    string
		secondary  string
		expected   []string
	}{
		{
			name:       "controller nameserver only",
			controller: "10.0.0.1",
			expected:   []string{"10.0.0.1"},
		},
		{
			name:       "appends primary and secondary upstreams after controller",
			controller: "10.0.0.1",
			primary:    "10.96.0.10",
			secondary:  "8.8.8.8",
			expected:   []string{"10.0.0.1", "10.96.0.10", "8.8.8.8"},
		},
		{
			name:       "ignores empty and duplicate upstreams",
			controller: "10.0.0.1",
			primary:    " 10.0.0.1 ",
			secondary:  "8.8.8.8",
			expected:   []string{"10.0.0.1", "8.8.8.8"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testLocalDNSProxyUpstreamGetter{
				primary:   tc.primary,
				secondary: tc.secondary,
			}

			actual := getLocalDNSProxyNameservers(cfg, tc.controller)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
