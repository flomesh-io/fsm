package fs

import (
	"path"

	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/bpf"
)

func GetPinningFile(objName string) string {
	return path.Join(BPFFSPath, bpf.FSM_PROG_NAME, objName)
}
