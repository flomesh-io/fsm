package maps

import (
	"unsafe"

	"github.com/cilium/ebpf"

	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/bpf"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/fs"
)

func GetAclEntries() map[AclKey]AclVal {
	items := make(map[AclKey]AclVal)
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_ACL)
	aclMap, mapErr := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{})
	if mapErr != nil {
		log.Fatal().Err(mapErr).Msgf("failed to load ebpf map: %s", pinnedFile)
	}
	defer aclMap.Close()
	aclKey := new(AclKey)
	aclVal := new(AclVal)
	it := aclMap.Iterate()
	for it.Next(unsafe.Pointer(aclKey), unsafe.Pointer(aclVal)) {
		items[*aclKey] = *aclVal
	}
	return items
}

func AddAclEntries(aclKeys []AclKey, aclVals []AclVal) (int, error) {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_ACL)
	if aclMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer aclMap.Close()
		return aclMap.BatchUpdate(aclKeys, aclVals, &ebpf.BatchOptions{})
	} else {
		return 0, err
	}
}

func DelAclEntries(aclKeys []AclKey) (int, error) {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_ACL)
	if aclMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer aclMap.Close()
		return aclMap.BatchDelete(aclKeys, &ebpf.BatchOptions{})
	} else {
		return 0, err
	}
}
