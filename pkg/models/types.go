package models

import (
	"time"

	"github.com/google/uuid"

	"github.com/flomesh-io/fsm/pkg/identity"
)

// TypeURI is a string describing the sidecar xDS payload.
type TypeURI string

func (t TypeURI) String() string {
	return string(t)
}

// ProxyKind is the type used to define the proxy's kind
type ProxyKind string

const (
	// KindSidecar implies the proxy is a sidecar
	KindSidecar ProxyKind = "sidecar"
)

// Proxy is an interface providing adaptiving proxies of multiple sidecars
type Proxy interface {
	GetUUID() uuid.UUID
	GetIdentity() identity.ServiceIdentity
	GetPodName() string
	GetPodNamespace() string
	GetConnectedAt() time.Time
}
