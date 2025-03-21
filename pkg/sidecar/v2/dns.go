package v2

import (
	"math"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/maps"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/util"
)

func (s *Server) xNetDNSProxyUpstreamsObserveFilter(obj interface{}) bool {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return false
	}
	upstreams := s.cfg.GetXNetDNSProxyUpstreams()
	for _, upstream := range upstreams {
		if strings.EqualFold(upstream.Name, service.Name) {
			if len(upstream.Namespace) > 0 {
				return strings.EqualFold(upstream.Namespace, service.Namespace)
			} else {
				return strings.EqualFold(s.cfg.GetFSMNamespace(), service.Namespace)
			}
		}
	}
	return false
}

func (s *Server) updateDNSNat() {
	obsoleteNats := make(map[string]*XNat)
	for natKey, natVal := range s.xnatCache {
		if natVal.key.Sys == uint32(maps.SysMesh) && natVal.key.TcDir == uint8(maps.TC_DIR_EGR) {
			obsoleteNats[natKey] = natVal
		}
	}

	brVal := s.getCniBridge4Info()

	natKey := new(maps.NatKey)
	natKey.Dport = util.HostToNetShort(53)
	natKey.Proto = uint8(maps.IPPROTO_UDP)
	natKey.TcDir = uint8(maps.TC_DIR_EGR)
	natVal := new(maps.NatVal)

	var upstreams []configv1alpha3.DNSUpstream

	if s.cfg.IsXNetDNSProxyEnabled() {
		upstreams = s.cfg.GetXNetDNSProxyUpstreams()
	}

	if len(upstreams) == 0 && s.cfg.IsLocalDNSProxyEnabled() {
		upstreams = append(upstreams, configv1alpha3.DNSUpstream{
			Name:      constants.FSMControllerName,
			Namespace: s.cfg.GetFSMNamespace(),
		})
	}

	if len(upstreams) > 0 {
		for _, upstream := range upstreams {
			var rip string
			if len(upstream.IP) > 0 {
				rip = upstream.IP
			} else if len(upstream.Name) > 0 {
				meshSvc := service.MeshService{
					Name:      upstream.Name,
					Namespace: upstream.Namespace,
				}
				if len(meshSvc.Namespace) == 0 {
					meshSvc.Namespace = s.cfg.GetFSMNamespace()
				}
				if k8sSvc := s.kubeController.GetService(meshSvc); k8sSvc != nil {
					rip = k8sSvc.Spec.ClusterIP
				}
			}
			if len(rip) == 0 {
				continue
			}

			rport := uint16(53)
			if upstream.Port > 0 && upstream.Port <= math.MaxUint16 {
				rport = uint16(upstream.Port)
			}
			natVal.AddEp(net.ParseIP(rip), rport, brVal.Mac[:], 0, 0, nil, true)
		}
	}

	if natVal.EpCnt > 0 {
		dnsNat := newXNat(maps.SysMesh, natKey, natVal)
		if existsNat, exists := s.xnatCache[dnsNat.Key()]; !exists {
			if err := s.setupDnsNat(natKey, natVal); err != nil {
				log.Error().Err(err).Msg(`failed to store dns nat`)
			}
			s.xnatCache[dnsNat.Key()] = dnsNat
		} else {
			if existsNat.valHash != dnsNat.valHash {
				if err := s.setupDnsNat(natKey, natVal); err != nil {
					log.Error().Err(err).Msg(`failed to store dns nat`)
				}
				s.xnatCache[dnsNat.Key()] = dnsNat
			}
			delete(obsoleteNats, dnsNat.Key())
		}
	}

	if len(obsoleteNats) > 0 {
		for natKey, xnat := range obsoleteNats {
			if err := s.unsetDnsNat(&xnat.key); err != nil {
				log.Error().Err(err).Msgf(`failed to unset dns nat`)
				continue
			}
			delete(s.xnatCache, natKey)
		}
	}
}

func (s *Server) setupDnsNat(natKey *maps.NatKey, natVal *maps.NatVal) error {
	if cfgVal, err := maps.GetXNetCfg(maps.SysMesh); err != nil {
		log.Fatal().Err(err).Msg(`failed to get xnet config`)
		return err
	} else {
		if !cfgVal.IPv4().IsSet(maps.CfgFlagOffsetUDPProtoAllowNatEscape) ||
			!cfgVal.IPv4().IsSet(maps.CfgFlagOffsetUDPNatByPortOn) ||
			cfgVal.IPv4().IsSet(maps.CfgFlagOffsetUDPProtoAllowAll) {
			cfgVal.IPv4().Clear(maps.CfgFlagOffsetUDPProtoAllowAll)
			cfgVal.IPv4().Set(maps.CfgFlagOffsetUDPProtoAllowNatEscape)
			cfgVal.IPv4().Set(maps.CfgFlagOffsetUDPNatByPortOn)
			if err = maps.SetXNetCfg(maps.SysMesh, cfgVal); err != nil {
				log.Fatal().Err(err).Msg(`failed to store xnet config`)
				return err
			}
		}
	}

	if err := maps.AddNatEntry(maps.SysMesh, natKey, natVal); err != nil {
		log.Fatal().Err(err).Msg(`failed to store dns nat`)
		return err
	}

	return nil
}

func (s *Server) unsetDnsNat(natKey *maps.NatKey) error {
	if cfgVal, err := maps.GetXNetCfg(maps.SysMesh); err != nil {
		log.Fatal().Err(err).Msg(`failed to get xnet config`)
		return err
	} else {
		if cfgVal.IPv4().IsSet(maps.CfgFlagOffsetUDPProtoAllowAll) ||
			!cfgVal.IPv4().IsSet(maps.CfgFlagOffsetUDPProtoAllowNatEscape) ||
			!cfgVal.IPv4().IsSet(maps.CfgFlagOffsetUDPNatByPortOn) {
			cfgVal.IPv4().Set(maps.CfgFlagOffsetUDPProtoAllowAll)
			cfgVal.IPv4().Clear(maps.CfgFlagOffsetUDPProtoAllowNatEscape)
			cfgVal.IPv4().Clear(maps.CfgFlagOffsetUDPNatByPortOn)
			if err = maps.SetXNetCfg(maps.SysMesh, cfgVal); err != nil {
				log.Fatal().Err(err).Msg(`failed to store xnet config`)
				return err
			}
		}
	}

	for _, tcDir := range []maps.TcDir{maps.TC_DIR_EGR} {
		natKey.TcDir = uint8(tcDir)
		if err := maps.DelNatEntry(maps.SysMesh, natKey); err != nil {
			log.Fatal().Err(err).Msg(`failed to store dns nat`)
			return err
		}
	}

	return nil
}
