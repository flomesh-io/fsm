package maps

import (
	"net"
	"unsafe"

	"github.com/cilium/ebpf"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/bpf"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/fs"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/util"
)

func AddNatEntry(natKey *NatKey, natVal *NatVal) error {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_NAT)
	if natMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer natMap.Close()
		if natVal.EpCnt > 0 {
			return natMap.Update(unsafe.Pointer(natKey), unsafe.Pointer(natVal), ebpf.UpdateAny)
		}
		err = natMap.Delete(unsafe.Pointer(natKey))
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	} else {
		return err
	}
}

func DelNatEntry(natKey *NatKey) error {
	pinnedFile := fs.GetPinningFile(bpf.FSM_MAP_NAME_NAT)
	if natMap, err := ebpf.LoadPinnedMap(pinnedFile, &ebpf.LoadPinOptions{}); err == nil {
		defer natMap.Close()
		err = natMap.Delete(unsafe.Pointer(natKey))
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	} else {
		return err
	}
}

func (t *NatVal) AddEp(raddr net.IP, rport uint16, rmac []uint8, inactive bool) (bool, error) {
	ipNb, err := util.IPv4ToInt(raddr)
	if err != nil {
		return false, err
	}
	portBe := util.HostToNetShort(rport)
	if t.EpCnt > 0 {
		for idx := range t.Eps {
			if t.Eps[idx].Raddr[0] == ipNb && t.Eps[idx].Rport == portBe {
				for n := range t.Eps[idx].Rmac {
					t.Eps[idx].Rmac[n] = rmac[n]
				}
				if inactive {
					t.Eps[idx].Inactive = 1
				} else {
					t.Eps[idx].Inactive = 0
				}
				return true, nil
			}
		}
	}

	if t.EpCnt >= uint16(len(t.Eps)) {
		return false, nil
	}

	t.Eps[t.EpCnt].Raddr[0] = ipNb
	t.Eps[t.EpCnt].Rport = portBe
	for n := range t.Eps[t.EpCnt].Rmac {
		t.Eps[t.EpCnt].Rmac[n] = rmac[n]
	}
	if inactive {
		t.Eps[t.EpCnt].Inactive = 1
	} else {
		t.Eps[t.EpCnt].Inactive = 0
	}
	t.EpCnt++
	return true, nil
}

func (t *NatVal) DelEp(raddr net.IP, rport uint16) error {
	ipNb, err := util.IPv4ToInt(raddr)
	if err != nil {
		return err
	}

	if t.EpCnt == 0 {
		return nil
	}

	portBe := util.HostToNetShort(rport)
	hitIdx := -1
	lastIdx := int(t.EpCnt - 1)

	for idx := range t.Eps {
		if t.Eps[idx].Raddr[0] == ipNb && t.Eps[idx].Rport == portBe {
			hitIdx = idx
			break
		}
	}

	if hitIdx == -1 {
		return nil
	}

	if hitIdx == lastIdx {
		t.Eps[hitIdx].Raddr[0] = 0
		t.Eps[hitIdx].Rport = 0
		t.Eps[hitIdx].Inactive = 0
	} else {
		t.Eps[hitIdx].Raddr[0] = t.Eps[lastIdx].Raddr[0]
		t.Eps[hitIdx].Rport = t.Eps[lastIdx].Rport
		t.Eps[hitIdx].Inactive = t.Eps[lastIdx].Inactive

		t.Eps[lastIdx].Raddr[0] = 0
		t.Eps[lastIdx].Rport = 0
		t.Eps[lastIdx].Inactive = 0
	}

	t.EpCnt--

	return nil
}
