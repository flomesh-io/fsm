// Package plugin implements ecnet cni plugin.
package plugin

import (
	"os"

	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("cni-plugin")
)

func init() {
	if logfile, err := os.OpenFile("/tmp/fsm-cni.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); err == nil {
		log = log.Output(logfile)
	}
}
