package maps

import (
	"unsafe"

	"github.com/cilium/ebpf"

	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/bpf"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/fs"
)

func GetXNetCfg(sysId SysID) (*CfgVal, error) {
	cfgVal := new(CfgVal)
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_CFG)
	if cfgMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer cfgMap.Close()
		cfgKey := CfgKey(sysId)
		err = cfgMap.Lookup(unsafe.Pointer(&cfgKey), unsafe.Pointer(cfgVal))
		return cfgVal, err
	} else {
		return nil, err
	}
}

func SetXNetCfg(sysId SysID, cfgVal *CfgVal) error {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_CFG)
	if cfgMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer cfgMap.Close()
		cfgKey := CfgKey(sysId)
		return cfgMap.Update(unsafe.Pointer(&cfgKey), unsafe.Pointer(cfgVal), ebpf.UpdateAny)
	} else {
		return err
	}
}

func (t *CfgVal) IPv4() *FlagT {
	return &t.Ipv4
}

func (t *CfgVal) IPv6() *FlagT {
	return &t.Ipv6
}

func (t *FlagT) Get(bit uint8) uint8 {
	bitMask := t.Flags >> bit
	return uint8(bitMask & 0x1)
}

func (t *FlagT) Set(bit uint8) {
	bitMask := uint64(1 << bit)
	t.Flags |= bitMask
}

func (t *FlagT) IsSet(bit uint8) bool {
	bitMask := t.Flags >> bit
	return uint8(bitMask&0x1) == 1
}

func (t *FlagT) Clear(bit uint8) {
	bitMask := uint64(1 << bit)
	t.Flags &= ^bitMask
}
