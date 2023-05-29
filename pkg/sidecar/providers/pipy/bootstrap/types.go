// Package bootstrap implements functionality related to Pipy's bootstrap config.
package bootstrap

import (
	"github.com/flomesh-io/fsm/pkg/models"
)

// Builder is the type used to build the Pipy bootstrap config.
type Builder struct {
	// RepoHost is the hostname of the Pipy Repo to connect to
	RepoHost string

	// RepoPort is the port of the Pipy Repo to connect to
	RepoPort uint32

	// NodeID is the proxy's node ID
	NodeID string

	// TLSMinProtocolVersion is the minimum supported TLS protocol version
	TLSMinProtocolVersion string

	// TLSMaxProtocolVersion is the maximum supported TLS protocol version
	TLSMaxProtocolVersion string

	// CipherSuites is the list of cipher that TLS 1.0-1.2 supports
	CipherSuites []string

	// ECDHCurves is the list of ECDH curves it supports
	ECDHCurves []string

	OriginalHealthProbes models.HealthProbes
}
