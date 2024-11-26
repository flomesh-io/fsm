package driver

import (
	sidecarv1 "github.com/flomesh-io/fsm/pkg/sidecar/v1"
)

const (
	driverName = `pipy`
)

func init() {
	sidecarv1.Register(driverName, new(PipySidecarDriver))
}
