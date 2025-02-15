package v2

import (
	"encoding/json"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/maps"
)

type XNat struct {
	key     maps.NatKey
	val     maps.NatVal
	keyHash uint64
	valHash uint64
}

func (lb *XNat) Key() string {
	bytes, _ := json.Marshal(lb.key)
	return string(bytes)
}

func (lb *XNat) NatKeyHash() uint64 {
	hash, _ := hashstructure.Hash(lb.key, hashstructure.FormatV2,
		&hashstructure.HashOptions{
			ZeroNil:         true,
			IgnoreZeroValue: true,
			SlicesAsSets:    true,
		})
	return hash
}

func (lb *XNat) NatValHash() uint64 {
	hash, _ := hashstructure.Hash(lb.val, hashstructure.FormatV2,
		&hashstructure.HashOptions{
			ZeroNil:         true,
			IgnoreZeroValue: true,
			SlicesAsSets:    true,
		})
	return hash
}

func newXNat(sysId maps.SysID, natKey *maps.NatKey, natVal *maps.NatVal) *XNat {
	natKey.Sys = uint32(sysId)
	e4lbNat := XNat{
		key: *natKey,
		val: *natVal,
	}
	e4lbNat.keyHash = e4lbNat.NatKeyHash()
	e4lbNat.valHash = e4lbNat.NatValHash()
	return &e4lbNat
}

func (s *Server) loadNatEntries() error {
	natEntries, err := maps.ListNatEntries()
	if err == nil {
		for natKey, natVal := range natEntries {
			for n := uint16(0); n < natVal.EpCnt; n++ {
				if natVal.Eps[n].Active > 0 {
					xnat := newXNat(maps.SysID(natKey.Sys), &natKey, &natVal)
					s.xnatCache[xnat.Key()] = xnat
				}
			}
		}
	}
	return err
}
