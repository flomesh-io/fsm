package catalog

import (
	"time"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/multicluster"
	"github.com/flomesh-io/fsm/pkg/plugin"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/smi"
	"github.com/flomesh-io/fsm/pkg/ticker"
)

var (
	meshCataloger MeshCataloger
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(kubeController k8s.Controller,
	meshSpec smi.MeshSpec,
	certManager *certificate.Manager,
	policyController policy.Controller,
	pluginController plugin.Controller,
	multiclusterController multicluster.Controller,
	stop <-chan struct{},
	cfg configurator.Configurator,
	serviceProviders []service.Provider,
	endpointsProviders []endpoint.Provider,
	msgBroker *messaging.Broker) *MeshCatalog {
	meshCatalog := &MeshCatalog{
		serviceProviders:       serviceProviders,
		endpointsProviders:     endpointsProviders,
		meshSpec:               meshSpec,
		policyController:       policyController,
		pluginController:       pluginController,
		multiclusterController: multiclusterController,
		configurator:           cfg,
		certManager:            certManager,
		kubeController:         kubeController,
	}

	meshCataloger = meshCatalog

	// Start the Resync ticker to tick based on the resync interval.
	// Starting the resync ticker only starts the ticker config watcher which
	// internally manages the lifecycle of the ticker routine.
	resyncTicker := ticker.NewResyncTicker(msgBroker, 30*time.Second /* min resync interval */)
	resyncTicker.Start(stop, cfg.GetConfigResyncInterval())

	return meshCatalog
}

func GetMeshCataloger() MeshCataloger {
	return meshCataloger
}

// GetKubeController returns the kube controller instance handling the current cluster
func (mc *MeshCatalog) GetKubeController() k8s.Controller {
	return mc.kubeController
}

// GetTrustDomain returns the currently configured trust domain, ie: cluster.local
func (mc *MeshCatalog) GetTrustDomain() string {
	return mc.certManager.GetTrustDomain()
}
