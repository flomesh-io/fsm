// Package events implements the eventing framework to receive and relay kubernetes events, and a framework to
// publish events to the Kubernetes API server.
package events

import (
	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("kube-events")
)

// Kubernetes Fatal Event reasons
// Fatal events are prefixed with 'Fatal' to help the event recording framework to wait for fatal
// events to be recorded prior to aborting.
const (
	// InvalidCLIParameters signifies invalid CLI parameters
	InvalidCLIParameters = "FatalInvalidCLIParameters"

	// InitializationError signifies an error during initialization
	InitializationError = "FatalInitializationError"

	// InvalidCertificateManager signifies that the certificate manager is invalid
	InvalidCertificateManager = "FatalInvalidCertificateManager"

	// CertificateIssuanceFailure signifies that a request to issue a certificate failed
	CertificateIssuanceFailure = "FatalCertificateIssuanceFailure"
)

// PubSubMessage represents a common messages abstraction to pass through the PubSub interface
type PubSubMessage struct {
	Kind   announcements.Kind
	OldObj interface{}
	NewObj interface{}
}
