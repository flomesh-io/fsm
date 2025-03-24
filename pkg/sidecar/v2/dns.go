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

func (s *Server) xNetDnsProxyUpstreamsObserveFilter(obj interface{}) bool {
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

type NatBrVal struct {
	natVal *maps.NatVal
	brVal  *maps.IFaceVal
}

func (s *Server) initDnsUpstreams() []configv1alpha3.DNSUpstream {
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
	return upstreams
}

func (s *Server) initDnsNatKeys() map[*maps.NatKey]*NatBrVal {
	nats := make(map[*maps.NatKey]*NatBrVal)
	if br4Val := s.getCniBridge4Info(); br4Val != nil {
		nat4Key := new(maps.NatKey)
		nat4Key.Dport = util.HostToNetShort(53)
		nat4Key.Proto = uint8(maps.IPPROTO_UDP)
		nat4Key.TcDir = uint8(maps.TC_DIR_EGR)
		nat4Key.V6 = 0
		nat4Val := new(maps.NatVal)
		nats[nat4Key] = &NatBrVal{
			natVal: nat4Val,
			brVal:  br4Val,
		}
	}
	if br6Val := s.getCniBridge6Info(); br6Val != nil {
		nat6Key := new(maps.NatKey)
		nat6Key.Dport = util.HostToNetShort(53)
		nat6Key.Proto = uint8(maps.IPPROTO_UDP)
		nat6Key.TcDir = uint8(maps.TC_DIR_EGR)
		nat6Key.V6 = 1
		nat6Val := new(maps.NatVal)
		nats[nat6Key] = &NatBrVal{
			natVal: nat6Val,
			brVal:  br6Val,
		}
	}
	return nats
}

func (s *Server) updateDnsNat() {
	obsoleteNats := make(map[string]*XNat)
	for natKey, natVal := range s.xnatCache {
		if natVal.key.Sys == uint32(maps.SysMesh) && natVal.key.TcDir == uint8(maps.TC_DIR_EGR) {
			obsoleteNats[natKey] = natVal
		}
	}

	nats := s.initDnsNatKeys()
	if len(nats) == 0 {
		return
	}

	upstreams := s.initDnsUpstreams()
	if len(upstreams) == 0 {
		return
	}

	for _, upstream := range upstreams {
		var rips []string
		if len(upstream.IP) > 0 {
			rips = append(rips, upstream.IP)
		} else if len(upstream.Name) > 0 {
			meshSvc := service.MeshService{
				Name:      upstream.Name,
				Namespace: upstream.Namespace,
			}
			if len(meshSvc.Namespace) == 0 {
				meshSvc.Namespace = s.cfg.GetFSMNamespace()
			}
			if k8sSvc := s.kubeController.GetService(meshSvc); k8sSvc != nil {
				for _, clusterIP := range k8sSvc.Spec.ClusterIPs {
					if len(clusterIP) > 0 {
						rips = append(rips, clusterIP)
					}
				}
			}
		}
		if len(rips) == 0 {
			continue
		}

		rPort := uint16(53)
		if upstream.Port > 0 && upstream.Port <= math.MaxUint16 {
			rPort = uint16(upstream.Port)
		}
		for _, rip := range rips {
			for natKey, nat := range nats {
				if strings.Contains(rip, ":") {
					if natKey.V6 == 1 {
						nat.natVal.AddEp(net.ParseIP(rip), rPort, nat.brVal.Mac[:], 0, 0, nil, true)
					}
				} else {
					if natKey.V6 == 0 {
						nat.natVal.AddEp(net.ParseIP(rip), rPort, nat.brVal.Mac[:], 0, 0, nil, true)
					}
				}
			}
		}
	}

	for natKey, nat := range nats {
		if nat.natVal.EpCnt > 0 {
			dnsNat := newXNat(maps.SysMesh, natKey, nat.natVal)
			if existsNat, exists := s.xnatCache[dnsNat.Key()]; !exists {
				if err := s.setupDnsNat(natKey, nat.natVal); err != nil {
					log.Error().Err(err).Msg(`failed to store dns nat`)
				}
				s.xnatCache[dnsNat.Key()] = dnsNat
			} else {
				if existsNat.valHash != dnsNat.valHash {
					if err := s.setupDnsNat(natKey, nat.natVal); err != nil {
						log.Error().Err(err).Msg(`failed to store dns nat`)
					}
					s.xnatCache[dnsNat.Key()] = dnsNat
				}
				delete(obsoleteNats, dnsNat.Key())
			}
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
