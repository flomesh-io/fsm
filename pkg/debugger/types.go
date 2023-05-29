// Package debugger implements functionality to provide information to debug the control plane via the debug HTTP server.
package debugger

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha4"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

var log = logger.New("debugger")

// DebugConfig implements the DebugServer interface.
type DebugConfig struct {
	certDebugger        CertificateManagerDebugger
	meshCatalogDebugger MeshCatalogDebugger
	kubeConfig          *rest.Config
	kubeClient          kubernetes.Interface
	kubeController      k8s.Controller
	configurator        configurator.Configurator
	msgBroker           *messaging.Broker
}

// CertificateManagerDebugger is an interface with methods for debugging certificate issuance.
type CertificateManagerDebugger interface {
	// ListIssuedCertificates returns the current list of certificates in FSM's cache.
	ListIssuedCertificates() []*certificate.Certificate
}

// MeshCatalogDebugger is an interface with methods for debugging Mesh Catalog.
type MeshCatalogDebugger interface {
	// ListSMIPolicies lists the SMI policies detected by FSM.
	ListSMIPolicies() ([]*split.TrafficSplit, []identity.K8sServiceAccount, []*spec.HTTPRouteGroup, []*access.TrafficTarget)
}
