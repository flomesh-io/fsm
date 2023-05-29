package catalog

import (
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/identity"
)

// GetConfigurator converts private variable to public
func (mc *MeshCatalog) GetConfigurator() *configurator.Configurator {
	return &mc.configurator
}

// ListEndpointsForServiceIdentity converts private method to public
func (mc *MeshCatalog) ListEndpointsForServiceIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	return mc.listEndpointsForServiceIdentity(serviceIdentity)
}
