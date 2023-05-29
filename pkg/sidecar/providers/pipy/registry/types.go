package registry

import (
	"sync"

	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

var log = logger.New("proxy-registry")

var (
	lock             sync.Mutex
	connectedProxies sync.Map
)

// ProxyRegistry keeps track of Sidecar proxies as they connect and disconnect
// from the control plane.
type ProxyRegistry struct {
	ProxyServiceMapper

	msgBroker *messaging.Broker

	// Fire a inform to update proxies
	UpdateProxies func()
}

// A simple interface to release certificates. Created to abstract the certificate.Manager struct for testing purposes.
type certificateReleaser interface {
	ReleaseCertificate(key string)
}
