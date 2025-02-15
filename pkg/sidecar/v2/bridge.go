package v2

import (
	"time"

	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/maps"
)

var (
	cniBridge4Val, cniBridge6Val *maps.IFaceVal
)

func (s *Server) getCniBridge4Info() *maps.IFaceVal {
	if cniBridge4Val != nil {
		return cniBridge4Val
	}

	brKey := new(maps.IFaceKey)
	brKey.Len = uint8(len(s.cniBridge4))
	copy(brKey.Name[0:brKey.Len], s.cniBridge4)
	for {
		var err error
		cniBridge4Val, err = maps.GetIFaceEntry(brKey)
		if err != nil {
			log.Error().Err(err).Msg(`failed to get node bridge4 info`)
			time.Sleep(time.Second * 5)
			continue
		}
		if cniBridge4Val == nil {
			log.Error().Msg(`failed to get node bridge4 info`)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}
	return cniBridge4Val
}

func (s *Server) getCniBridge6Info() *maps.IFaceVal {
	if cniBridge6Val != nil {
		return cniBridge6Val
	}

	brKey := new(maps.IFaceKey)
	brKey.Len = uint8(len(s.cniBridge6))
	copy(brKey.Name[0:brKey.Len], s.cniBridge6)
	for {
		var err error
		cniBridge6Val, err = maps.GetIFaceEntry(brKey)
		if err != nil {
			log.Error().Err(err).Msg(`failed to get node bridge6 info`)
			time.Sleep(time.Second * 5)
			continue
		}
		if cniBridge6Val == nil {
			log.Error().Msg(`failed to get node bridge6 info`)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}
	return cniBridge6Val
}
