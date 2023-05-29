package driver

import (
	"github.com/flomesh-io/fsm/pkg/sidecar"
)

const (
	driverName = `pipy`
)

func init() {
	sidecar.Register(driverName, new(PipySidecarDriver))
}
