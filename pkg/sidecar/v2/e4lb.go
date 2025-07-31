package v2

import (
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilnet "k8s.io/utils/net"

	xnetv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/xnetwork/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/maps"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/neigh"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/route"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/util"
)

func (s *Server) doConfigE4LBs() {
	if s.Leading {
		s.doE4lbLayout()
	}
	s.doApplyE4LBs()
}

func (s *Server) doE4lbLayout() {
	topo := &e4lbTopo{
		nodeCache:         make(map[string]bool),
		nodeEipLayout:     make(map[string]map[string]uint8),
		eipNodeLayout:     make(map[string]string),
		eipSvcCache:       make(map[string]uint8),
		advertisementHash: make(map[types.UID]uint64),
	}
	availableNetworkNodes(s.kubeClient, topo)
	if eipAdvs := s.xnetworkController.GetEIPAdvertisements(); len(eipAdvs) > 0 {
		topo.loadEIPAdvertisements(eipAdvs)
		topo.processEIPAdvertisements(eipAdvs, s.xnetworkClient)
	}
}

func (s *Server) doApplyE4LBs() {
	e4lbSvcs := make(map[types.UID]*corev1.Service)
	e4lbEips := make(map[types.UID][]string)
	if eipAdvs := s.xnetworkController.GetEIPAdvertisements(); len(eipAdvs) > 0 {
		s.doApplyEIPAdvertisements(eipAdvs, e4lbSvcs, e4lbEips)
	}
	s.announceE4LBService(e4lbSvcs, e4lbEips)
}

func (s *Server) doApplyEIPAdvertisements(eipAdvs []*xnetv1alpha1.EIPAdvertisement, e4lbSvcs map[types.UID]*corev1.Service, e4lbEips map[types.UID][]string) {
	for _, eipAdv := range eipAdvs {
		if len(eipAdv.Status.Announce) == 0 {
			continue
		}
		meshSvc := service.MeshService{Name: eipAdv.Spec.Service.Name}
		if len(eipAdv.Spec.Service.Namespace) > 0 {
			meshSvc.Namespace = eipAdv.Spec.Service.Namespace
		} else {
			meshSvc.Namespace = eipAdv.Namespace
		}
		if k8sSvc := s.kubeController.GetService(meshSvc); k8sSvc != nil {
			var announceEips []string
			for eip, selectedNode := range eipAdv.Status.Announce {
				ipAddr := net.ParseIP(eip)
				if ipAddr == nil || (ipAddr.To4() == nil && ipAddr.To16() == nil) || ipAddr.IsUnspecified() || ipAddr.IsMulticast() {
					continue
				}
				if strings.EqualFold(selectedNode, s.nodeName) {
					announceEips = append(announceEips, eip)
				}
			}
			if len(announceEips) > 0 {
				e4lbSvcs[k8sSvc.GetUID()] = k8sSvc
				e4lbEips[k8sSvc.GetUID()] = announceEips
			}
		}
	}
}

func (s *Server) announceE4LBService(e4lbSvcs map[types.UID]*corev1.Service, e4lbEips map[types.UID][]string) {
	obsoleteNats := make(map[string]*XNat)
	for natKey, natVal := range s.xnatCache {
		if natVal.key.Sys == uint32(maps.SysE4lb) {
			obsoleteNats[natKey] = natVal
			log.Debug().Msgf("obsoleteNats natKey: %s", natKey)
		}
	}

	s.setupE4LBNats(e4lbSvcs, e4lbEips, obsoleteNats)

	log.Debug().Msgf("obsoleteNats left: %d", len(obsoleteNats))

	for natKey, e4lbNat := range obsoleteNats {
		log.Debug().Msgf("obsoleteNats left key: %s", natKey)
		if e4lbNat.key.Sys == uint32(maps.SysE4lb) {
			if err := s.unsetE4LBNodeNat(&e4lbNat.key); err == nil {
				delete(s.xnatCache, natKey)
			} else {
				log.Error().Err(err).Msgf(`failed to unset e4lb node nat`)
			}
		}
	}
}

func (s *Server) setupE4LBNats(e4lbSvcs map[types.UID]*corev1.Service, e4lbEips map[types.UID][]string, obsoleteNats map[string]*XNat) {
	defaultIfi, defaultEth, defaultHwAddr, err := s.discoverGateway()
	if err != nil {
		log.Error().Err(err).Msg(`fail to discover gateway`)
		return
	}

	obsoletes := s.eipCache.Items()
	for uid, k8sSvc := range e4lbSvcs {
		eips, exists := e4lbEips[uid]
		if !exists || len(eips) == 0 {
			continue
		}

		upstreams := make(map[string]bool)
		if k8s.IsHeadlessService(k8sSvc) {
			if microSvc := s.headlessService(k8sSvc, upstreams); !microSvc {
				continue
			}
		} else {
			for _, clusterIP := range k8sSvc.Spec.ClusterIPs {
				upstreams[clusterIP] = false
			}
		}

		if len(upstreams) == 0 || len(k8sSvc.Spec.Ports) == 0 {
			continue
		}

		for _, eip := range eips {
			for _, port := range k8sSvc.Spec.Ports {
				if !strings.EqualFold(string(port.Protocol), string(corev1.ProtocolTCP)) {
					continue
				}
				ePort := uint16(0)
				if port.Port > 0 && port.Port <= math.MaxUint16 {
					ePort = uint16(port.Port)
				}

				nodeNatKey, nodeNatVal := s.getE4lbNodeNat(maps.SysE4lb, eip, ePort, upstreams, port)
				for _, tcDir := range []maps.TcDir{maps.TC_DIR_IGR, maps.TC_DIR_EGR} {
					nodeNatKey.TcDir = uint8(tcDir)
					nodeNat := newXNat(nodeNatKey, nodeNatVal)
					log.Debug().Msgf("nodeNat.Key: %s", nodeNat.Key())
					if existsNat, exists := s.xnatCache[nodeNat.Key()]; !exists {
						if err := s.setupE4LBNodeNat(nodeNatKey, nodeNatVal); err == nil {
							log.Debug().Msgf("nodeNat.Key() cache add: %s", nodeNat.Key())
							s.xnatCache[nodeNat.Key()] = nodeNat
						} else {
							log.Error().Err(err).Msgf(`failed to setup e4lb node nat, eip: %s`, eip)
						}
					} else {
						if existsNat.valHash != nodeNat.valHash {
							if err := s.setupE4LBNodeNat(nodeNatKey, nodeNatVal); err == nil {
								log.Debug().Msgf("nodeNat.Key() cache add: %s", nodeNat.Key())
								s.xnatCache[nodeNat.Key()] = nodeNat
							} else {
								log.Error().Err(err).Msgf(`failed to setup e4lb node nat, eip: %s`, eip)
							}
						}
						log.Debug().Msgf("nodeNat.Key() obsoleteNats del: %s", nodeNat.Key())
						delete(obsoleteNats, nodeNat.Key())
					}
				}
			}

			if _, exists := obsoletes[eip]; exists {
				delete(obsoletes, eip)
			} else {
				s.eipCache.Set(eip, &e4lbNeigh{
					eip:     net.ParseIP(eip),
					ifName:  defaultEth,
					ifIndex: defaultIfi,
					macAddr: defaultHwAddr,
					adv:     false,
				})
			}
		}
	}

	for eip, n := range obsoletes {
		if n.adv {
			neigh.DelNeighOverIface(n.ifIndex, n.eip, n.macAddr)
		}
		s.eipCache.Remove(eip)
	}
}

func (s *Server) gratuitousEIPs() {
	eips := s.eipCache.Keys()
	for _, eip := range eips {
		if n, exists := s.eipCache.Get(eip); exists {
			n.adv = true
			neigh.SetNeighOverIface(n.ifIndex, n.eip, n.macAddr)
			neigh.GratuitousNeighOverIface(n.ifName, n.ifIndex, n.eip, n.macAddr)
		}
	}
}

func (s *Server) getE4lbNodeNat(sysId maps.SysID, eIP string, ePort uint16, upstreams map[string]bool, port corev1.ServicePort) (*maps.NatKey, *maps.NatVal) {
	eipAddr := net.ParseIP(eIP)
	natKey := new(maps.NatKey)
	natKey.Sys = uint32(sysId)
	natKey.Dport = util.HostToNetShort(ePort)
	natKey.Proto = uint8(maps.IPPROTO_TCP)
	natKey.Daddr[0], natKey.Daddr[1], natKey.Daddr[2], natKey.Daddr[3], natKey.V6, _ = util.IPToInt(eipAddr)

	natVal := new(maps.NatVal)
	for rip, microSvc := range upstreams {
		if natKey.V6 == 0 && !utilnet.IsIPv4String(rip) {
			continue
		}
		if natKey.V6 == 1 && !utilnet.IsIPv6String(rip) {
			continue
		}
		ripAddr := net.ParseIP(rip)
		if microSvc {
			if port.TargetPort.IntVal > 0 && port.TargetPort.IntVal <= math.MaxUint16 {
				rport := uint16(port.TargetPort.IntVal)
				iface, hwAddr, err := s.matchRoute(rip)
				if err != nil {
					continue
				}
				natVal.AddEp(ripAddr, rport, hwAddr[:], uint32(iface.Index), maps.BPF_F_EGRESS, nil, true)
			} else {
				continue
			}
		} else {
			rport := ePort
			var brVal *maps.IFaceVal
			if natKey.V6 == 1 {
				brVal = s.getCniBridge6Info()
			} else {
				brVal = s.getCniBridge4Info()
			}
			if brVal == nil {
				continue
			}
			natVal.AddEp(ripAddr, rport, brVal.Xmac[:], brVal.Ifi, maps.BPF_F_INGRESS, nil, true)
		}
	}
	return natKey, natVal
}

func (s *Server) headlessService(k8sSvc *corev1.Service, upstreams map[string]bool) bool {
	microSvc := false
	if len(k8sSvc.Annotations) > 0 {
		if _, ok := k8sSvc.Annotations[connector.AnnotationCloudServiceInheritedFrom]; ok {
			if v, exists := k8sSvc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
				microSvc = true
				microMeta := connector.Decode(k8sSvc, v)
				for addr := range microMeta.Endpoints {
					upstreams[string(addr)] = true
					for _, port := range k8sSvc.Spec.Ports {
						if corev1.ProtocolTCP == port.Protocol || corev1.ProtocolSCTP == port.Protocol {
							go func() {
								s.doTCPProbe(string(addr), strconv.Itoa(int(port.Port)), time.Second*3)
							}()
						}
						if corev1.ProtocolUDP == port.Protocol {
							go func() {
								s.doUDPProbe(string(addr), strconv.Itoa(int(port.Port)), time.Second*3)
							}()
						}
					}
				}
			}
		}
	}
	return microSvc
}

func (s *Server) discoverGateway() (int, string, net.HardwareAddr, error) {
	if dev, _, err := route.DiscoverGateway(); err != nil {
		return 0, "", nil, err
	} else if viaEth, err := netlink.LinkByName(dev); err != nil {
		return 0, "", nil, err
	} else {
		defaultHwAddr := viaEth.Attrs().HardwareAddr
		defaultEth := dev
		defaultIfi := viaEth.Attrs().Index
		return defaultIfi, defaultEth, defaultHwAddr, nil
	}
}

func (s *Server) setupE4LBNodeNat(natKey *maps.NatKey, natVal *maps.NatVal) error {
	return maps.AddNatEntry(maps.SysE4lb, natKey, natVal)
}

func (s *Server) unsetE4LBNodeNat(natKey *maps.NatKey) error {
	return maps.DelNatEntry(maps.SysE4lb, natKey)
}
