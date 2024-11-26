package registry

import (
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
)

// NewProxyRegistry initializes a new empty *ProxyRegistry.
func NewProxyRegistry(mapper ProxyServiceMapper, msgBroker *messaging.Broker) *ProxyRegistry {
	return &ProxyRegistry{
		ProxyServiceMapper: mapper,
		msgBroker:          msgBroker,
	}
}

// RegisterProxy registers a newly connected proxy.
func (pr *ProxyRegistry) RegisterProxy(proxy *pipy.Proxy) *pipy.Proxy {
	lock.Lock()
	defer lock.Unlock()
	actual, loaded := connectedProxies.LoadOrStore(proxy.UUID.String(), proxy)
	if loaded {
		return actual.(*pipy.Proxy)
	}
	log.Debug().Str("proxy", proxy.String()).Msg("Registered new proxy")
	return proxy
}

// GetConnectedProxy loads a connected proxy from the registry.
func (pr *ProxyRegistry) GetConnectedProxy(uuid string) *pipy.Proxy {
	lock.Lock()
	defer lock.Unlock()
	p, ok := connectedProxies.Load(uuid)
	if !ok {
		return nil
	}
	return p.(*pipy.Proxy)
}

// RangeConnectedProxy calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (pr *ProxyRegistry) RangeConnectedProxy(f func(key, value interface{}) bool) {
	connectedProxies.Range(f)
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (pr *ProxyRegistry) UnregisterProxy(p *pipy.Proxy) {
	p.Quit <- true
	connectedProxies.Delete(p.UUID.String())
	log.Debug().Msgf("Unregistered proxy %s", p.String())
}

// GetConnectedProxyCount counts the number of connected proxies
// TODO(steeling): switch to a regular map with mutex so we can get the count without iterating the entire list.
func (pr *ProxyRegistry) GetConnectedProxyCount() int {
	return len(pr.ListConnectedProxies())
}
