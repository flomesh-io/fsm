package v2

import (
	"net"
	"time"

	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/maps"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/util"
)

var dnsNatDone = false

func (s *Server) updateDNSNat() {
	var dnsClusterAddr string

	meshSvc := service.MeshService{
		Name:      `fsm-controller`,
		Namespace: s.cfg.GetFSMNamespace(),
	}
	if k8sSvc := s.kubeController.GetService(meshSvc); k8sSvc != nil {
		dnsClusterAddr = k8sSvc.Spec.ClusterIP
	}
	if len(dnsClusterAddr) == 0 {
		return
	}

	if s.cfg.IsLocalDNSProxyEnabled() && !dnsNatDone {
		s.setupDnsNat(dnsClusterAddr)
		dnsNatDone = true
	}

	if !s.cfg.IsLocalDNSProxyEnabled() && dnsNatDone {
		s.resetDnsNat()
		dnsNatDone = false
	}
}

func (s *Server) setupDnsNat(dnsAddr string) {
	var err error
	var brVal *maps.IFaceVal
	var cfgVal *maps.CfgVal
	brKey := new(maps.IFaceKey)
	brKey.Len = uint8(len(bridgeDev))
	copy(brKey.Name[0:brKey.Len], bridgeDev)
	for {
		brVal, err = maps.GetIFaceEntry(brKey)
		if err != nil {
			log.Error().Err(err).Msg(`failed to get node bridge info`)
			time.Sleep(time.Second * 5)
			continue
		}
		if brVal == nil {
			log.Error().Msg(`failed to get node bridge info`)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}

	if cfgVal, err = maps.GetXNetCfg(); err != nil {
		log.Fatal().Err(err).Msg(`failed to get xnet config`)
	} else {
		cfgVal.Clear(maps.CfgFlagOffsetIPv4UDPProtoAllowAll)
		cfgVal.Set(maps.CfgFlagOffsetIPv4UDPProtoAllowNatEscape)
		cfgVal.Set(maps.CfgFlagOffsetIPv4UDPNatByPortOn)
		if err = maps.SetXNetCfg(cfgVal); err != nil {
			log.Fatal().Err(err).Msg(`failed to store xnet config`)
		}
	}

	natKey := new(maps.NatKey)
	natKey.Dport = util.HostToNetShort(53)
	natKey.Proto = uint8(maps.IPPROTO_UDP)
	natVal := new(maps.NatVal)
	natVal.AddEp(net.ParseIP(dnsAddr), 53, brVal.Mac[:], false)
	for _, tcDir := range []maps.TcDir{maps.TC_DIR_IGR, maps.TC_DIR_EGR} {
		natKey.TcDir = uint8(tcDir)
		if err = maps.AddNatEntry(natKey, natVal); err != nil {
			log.Fatal().Err(err).Msg(`failed to store dns nat`)
		}
	}
}

func (s *Server) resetDnsNat() {
	if cfgVal, err := maps.GetXNetCfg(); err != nil {
		log.Fatal().Err(err).Msg(`failed to get xnet config`)
	} else {
		cfgVal.Set(maps.CfgFlagOffsetIPv4UDPProtoAllowAll)
		cfgVal.Clear(maps.CfgFlagOffsetIPv4UDPProtoAllowNatEscape)
		cfgVal.Clear(maps.CfgFlagOffsetIPv4UDPNatByPortOn)
		if err = maps.SetXNetCfg(cfgVal); err != nil {
			log.Fatal().Err(err).Msg(`failed to store xnet config`)
		}
	}

	natKey := new(maps.NatKey)
	natKey.Dport = util.HostToNetShort(53)
	natKey.Proto = uint8(maps.IPPROTO_UDP)
	for _, tcDir := range []maps.TcDir{maps.TC_DIR_IGR, maps.TC_DIR_EGR} {
		natKey.TcDir = uint8(tcDir)
		if err := maps.DelNatEntry(natKey); err != nil {
			log.Fatal().Err(err).Msg(`failed to store dns nat`)
		}
	}
}
