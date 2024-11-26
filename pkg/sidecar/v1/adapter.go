// Package sidecar implements adapter's methods.
package v1

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/models"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/driver"
)

var (
	driversMutex sync.RWMutex
	drivers      = make(map[string]driver.Driver)
	engineDriver driver.Driver
)

// NewCertCNPrefix returns a newly generated CommonName for a certificate of the form: <ProxyUUID>.<kind>.<identity>
// where identity itself is of the form <name>.<namespace>
func NewCertCNPrefix(proxyUUID uuid.UUID, kind models.ProxyKind, si identity.ServiceIdentity) string {
	return fmt.Sprintf("%s.%s.%s", proxyUUID.String(), kind, si.String())
}

// GetCertCNPrefix returns a newly generated CommonName for a certificate of the form: <ProxyUUID>.<kind>.<identity>
// where identity itself is of the form <name>.<namespace>
func GetCertCNPrefix(proxy models.Proxy, kind models.ProxyKind) string {
	return fmt.Sprintf("%s.%s.%s", proxy.GetUUID().String(), kind, proxy.GetIdentity())
}

// InstallDriver is to serve as an indication of the using sidecar driver
func InstallDriver(driverName string) error {
	driversMutex.Lock()
	defer driversMutex.Unlock()
	registeredDriver, ok := drivers[driverName]
	if !ok {
		return fmt.Errorf("sidecar: unknown driver %q (forgot to import?)", driverName)
	}
	engineDriver = registeredDriver
	return nil
}

// Register makes a sidecar driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.Driver) {
	driversMutex.Lock()
	defer driversMutex.Unlock()
	if driver == nil {
		panic("sidecar: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("sidecar: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Patch is an adapter method for InjectorDriver.Patch
func Patch(ctx context.Context) error {
	driversMutex.RLock()
	defer driversMutex.RUnlock()
	if engineDriver == nil {
		return errors.New("sidecar: unknown driver (forgot to init?)")
	}
	return engineDriver.Patch(ctx)
}

// Start is an adapter method for ControllerDriver.Start
func Start(ctx context.Context) (health.Probes, error) {
	driversMutex.RLock()
	defer driversMutex.RUnlock()
	if engineDriver == nil {
		return nil, errors.New("sidecar: unknown driver (forgot to init?)")
	}
	return engineDriver.Start(ctx)
}
