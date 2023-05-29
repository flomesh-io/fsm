package debugger

import (
	"net/http"
	"net/http/pprof"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// GetHandlers implements DebugConfig interface and returns the rest of URLs and the handling functions.
func (ds DebugConfig) GetHandlers(handlers map[string]http.Handler) map[string]http.Handler {
	for url, handler := range map[string]http.Handler{
		"/debug/certs":         ds.getCertHandler(),
		"/debug/policies":      ds.getSMIPoliciesHandler(),
		"/debug/config":        ds.getFSMConfigHandler(),
		"/debug/namespaces":    ds.getMonitoredNamespacesHandler(),
		"/debug/feature-flags": ds.getFeatureFlags(),

		// Pprof handlers
		"/debug/pprof/":        http.HandlerFunc(pprof.Index),
		"/debug/pprof/cmdline": http.HandlerFunc(pprof.Cmdline),
		"/debug/pprof/profile": http.HandlerFunc(pprof.Profile),
		"/debug/pprof/symbol":  http.HandlerFunc(pprof.Symbol),
		"/debug/pprof/trace":   http.HandlerFunc(pprof.Trace),
	} {
		handlers[url] = handler
	}

	// provides an index of the available /debug endpoints
	handlers["/debug"] = ds.getDebugIndex(handlers)

	return handlers
}

// NewDebugConfig returns an implementation of DebugConfig interface.
func NewDebugConfig(certDebugger CertificateManagerDebugger, meshCatalogDebugger MeshCatalogDebugger,
	kubeConfig *rest.Config, kubeClient kubernetes.Interface,
	cfg configurator.Configurator, kubeController k8s.Controller, msgBroker *messaging.Broker) DebugConfig {
	return DebugConfig{
		certDebugger:        certDebugger,
		meshCatalogDebugger: meshCatalogDebugger,
		kubeClient:          kubeClient,
		kubeController:      kubeController,

		// We need the Kubernetes config to be able to establish port forwarding to the Sidecar pod we want to debug.
		kubeConfig: kubeConfig,

		configurator: cfg,

		msgBroker: msgBroker,
	}
}
