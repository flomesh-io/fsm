package fs

import (
	"path"
	"time"

	"github.com/cilium/ebpf/rlimit"

	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/util"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/volume"
)

var (
	BPFFSPath = `/sys/fs/bpf`

	log = logger.New("fsm-xnet-bpf-fs")
)

func init() {
	for !util.Exists(volume.Sysfs.MountPath) {
		time.Sleep(time.Second * 2)
	}

	BPFFSPath = path.Join(volume.Sysfs.MountPath, `bpf`)

	if err := rlimit.RemoveMemlock(); err != nil {
		log.Error().Msgf("remove mem lock error: %v", err)
	}
}
