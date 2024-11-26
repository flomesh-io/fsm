package maps

import (
	"unsafe"

	"github.com/cilium/ebpf"

	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/bpf"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/fs"
)

func AddIFaceEntry(ifaceKey *IFaceKey, ifaceVal *IFaceVal) error {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_IFS)
	if ifaceMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer ifaceMap.Close()
		return ifaceMap.Update(unsafe.Pointer(ifaceKey), unsafe.Pointer(ifaceVal), ebpf.UpdateAny)
	} else {
		return err
	}
}

func GetIFaceEntry(ifaceKey *IFaceKey) (*IFaceVal, error) {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_IFS)
	if natMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer natMap.Close()
		ifaceVal := new(IFaceVal)
		err = natMap.Lookup(unsafe.Pointer(ifaceKey), unsafe.Pointer(ifaceVal))
		return ifaceVal, err
	} else {
		return nil, err
	}
}
