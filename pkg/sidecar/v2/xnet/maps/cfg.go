package maps

import (
	"unsafe"

	"github.com/cilium/ebpf"

	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/bpf"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/fs"
)

func GetXNetCfg() (*CfgVal, error) {
	cfgVal := new(CfgVal)
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_CFG)
	if cfgMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer cfgMap.Close()
		cfgKey := CfgKey(0)
		err = cfgMap.Lookup(unsafe.Pointer(&cfgKey), unsafe.Pointer(cfgVal))
		return cfgVal, err
	} else {
		return nil, err
	}
}

func SetXNetCfg(cfgVal *CfgVal) error {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_CFG)
	if cfgMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer cfgMap.Close()
		cfgKey := CfgKey(0)
		return cfgMap.Update(unsafe.Pointer(&cfgKey), unsafe.Pointer(cfgVal), ebpf.UpdateAny)
	} else {
		return err
	}
}

func (t *CfgVal) Get(bit uint8) uint8 {
	bitMask := t.Flags >> bit
	return uint8(bitMask & 0x1)
}

func (t *CfgVal) Set(bit uint8) {
	bitMask := uint64(1 << bit)
	t.Flags |= bitMask
}

func (t *CfgVal) IsSet(bit uint8) bool {
	bitMask := t.Flags >> bit
	return uint8(bitMask&0x1) == 1
}

func (t *CfgVal) Clear(bit uint8) {
	bitMask := uint64(1 << bit)
	t.Flags &= ^bitMask
}
