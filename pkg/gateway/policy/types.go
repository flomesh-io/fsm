package policy

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("fsm-gateway/policy")
)

type UpstreamTLSConfig struct {
	MTLS   *bool
	Secret *corev1.Secret
}
