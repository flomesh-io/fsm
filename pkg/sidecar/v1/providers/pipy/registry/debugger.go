package registry

import (
	"github.com/flomesh-io/fsm/pkg/models"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
)

// ListConnectedProxies lists the Pipy proxies already connected and the time they first connected.
func (pr *ProxyRegistry) ListConnectedProxies() map[string]models.Proxy {
	proxies := make(map[string]models.Proxy)
	connectedProxies.Range(func(keyIface, propsIface interface{}) bool {
		uuid := keyIface.(string)
		proxies[uuid] = propsIface.(*pipy.Proxy)
		return true // continue the iteration
	})
	return proxies
}
