package v2

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
)

// ---

type RetryPolicyProcessor struct {
	generator *ConfigGenerator
}

func NewRetryPolicyProcessor(c *ConfigGenerator) BackendPolicyProcessor {
	return &RetryPolicyProcessor{
		generator: c,
	}
}

func (p RetryPolicyProcessor) Process(route client.Object, backendRef gwv1.BackendObjectReference, svcPort *v2.ServicePortName) {
	//TODO implement me
	panic("implement me")
}
