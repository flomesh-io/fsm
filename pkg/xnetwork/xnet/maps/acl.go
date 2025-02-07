package maps

import (
	"unsafe"

	"github.com/cilium/ebpf"

	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/bpf"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/fs"
)

func GetAclEntries(sysId SysID) map[AclKey]AclVal {
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
		if aclKey.Sys == uint32(sysId) {
			items[*aclKey] = *aclVal
		}
	}
	return items
}

func AddAclEntries(sysId SysID, aclKeys []AclKey, aclVals []AclVal) (int, error) {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_ACL)
	if aclMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer aclMap.Close()
		for idx := range aclKeys {
			aclKeys[idx].Sys = uint32(sysId)
		}
		return aclMap.BatchUpdate(aclKeys, aclVals, &ebpf.BatchOptions{})
	} else {
		return 0, err
	}
}

func DelAclEntries(sysId SysID, aclKeys []AclKey) (int, error) {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_ACL)
	if aclMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer aclMap.Close()
		for idx := range aclKeys {
			aclKeys[idx].Sys = uint32(sysId)
		}
		return aclMap.BatchDelete(aclKeys, &ebpf.BatchOptions{})
	} else {
		return 0, err
	}
}
