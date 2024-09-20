package dns

import (
	"fmt"

	"github.com/jonboulle/clockwork"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/configurator"
)

// WallClock is the wall clock
var WallClock = clockwork.NewRealClock()

// Config holds the configuration parameters
type Config struct {
	cfg              configurator.Configurator
	CustomDNSRecords []string // manual custom dns entries
}

// GetNameservers nameservers to forward queries to
func (c *Config) GetNameservers() []string {
	var nameservers []string
	if upstream := c.cfg.GetLocalDNSProxyPrimaryUpstream(); len(upstream) > 0 {
		nameservers = append(nameservers, fmt.Sprintf("%s:53", upstream))
	}
	if upstream := c.cfg.GetLocalDNSProxySecondaryUpstream(); len(upstream) > 0 {
		nameservers = append(nameservers, fmt.Sprintf("%s:53", upstream))
	}
	return nameservers
}

func (c *Config) IsWildcard() bool {
	return c.cfg.IsWildcardDNSProxyEnabled()
}

func (c *Config) GetWildcardResolveDB() []configv1alpha3.ResolveAddr {
	return c.cfg.GetMeshConfig().Spec.Sidecar.LocalDNSProxy.Wildcard.IPs
}

func (c *Config) GetLoopbackResolveDB() []configv1alpha3.ResolveAddr {
	return c.cfg.GetMeshConfig().Spec.Sidecar.LocalDNSProxy.Wildcard.LOs
}

func (c *Config) GenerateIPv6BasedOnIPv4() bool {
	return c.cfg.GenerateIPv6BasedOnIPv4()
}

// GetNXDomain response to blocked queries with a NXDOMAIN
func (c *Config) GetNXDomain() bool {
	return false
}

// GetInterval concurrency interval for lookups in miliseconds
func (c *Config) GetInterval() int {
	return 200
}

// GetTimeout query timeout for dns lookups in seconds
func (c *Config) GetTimeout() int {
	return 5
}
