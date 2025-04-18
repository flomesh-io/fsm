package v2

import (
	"context"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	xnetv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/xnetwork/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/arp"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/maps"
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
	topo := &E4lbTopo{
		NodeCache:       make(map[string]bool),
		NodeEipLayout:   make(map[string]map[string]uint8),
		EipNodeLayout:   make(map[string]string),
		EipSvcCache:     make(map[string]uint8),
		AdvAnnounceHash: make(map[types.UID]uint64),
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
		} else if natVal.key.Sys == uint32(maps.SysMesh) &&
			natVal.key.Proto == uint8(maps.IPPROTO_TCP) &&
			natVal.key.TcDir == uint8(maps.TC_DIR_EGR) {
			if natVal.key.Daddr[0] == 0 && natVal.key.Daddr[1] == 0 &&
				natVal.key.Daddr[2] == 0 && natVal.key.Daddr[3] == 0 && natVal.key.Dport == 0 {
				continue
			}
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
		} else if e4lbNat.key.Sys == uint32(maps.SysMesh) {
			if err := s.unsetE4LBPodNat(&e4lbNat.key); err == nil {
				delete(s.xnatCache, natKey)
			} else {
				log.Error().Err(err).Msgf(`failed to unset e4lb pod nat`)
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

				s.setupE4LBServiceNeigh(defaultIfi, eip, defaultHwAddr)

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

				podNatKey, podNatVal := s.getE4lbPodNat(maps.SysMesh, eip, ePort, upstreams, port)
				for _, tcDir := range []maps.TcDir{maps.TC_DIR_EGR} {
					podNatKey.TcDir = uint8(tcDir)
					podNat := newXNat(podNatKey, podNatVal)
					log.Debug().Msgf("podNat.Key: %s", podNat.Key())
					if existsNat, exists := s.xnatCache[podNat.Key()]; !exists {
						if err := s.setupE4LBPodNat(podNatKey, podNatVal); err == nil {
							log.Debug().Msgf("podNat.Key() cache add: %s", podNat.Key())
							s.xnatCache[podNat.Key()] = podNat
						} else {
							log.Error().Err(err).Msgf(`failed to setup e4lb pod nat, eip: %s`, eip)
						}
					} else {
						if existsNat.valHash != podNat.valHash {
							if err := s.setupE4LBPodNat(podNatKey, podNatVal); err == nil {
								log.Debug().Msgf("podNat.Key() cache add: %s", podNat.Key())
								s.xnatCache[podNat.Key()] = podNat
							} else {
								log.Error().Err(err).Msgf(`failed to setup e4lb pod nat, eip: %s`, eip)
							}
						}
						log.Debug().Msgf("podNat.Key() obsoleteNats del: %s", podNat.Key())
						delete(obsoleteNats, podNat.Key())
					}
				}
			}

			if err := arp.Announce(defaultEth, eip, defaultHwAddr); err != nil {
				log.Error().Msg(err.Error())
			}
		}
	}
}

func (s *Server) getE4lbNodeNat(sysId maps.SysID, eIP string, ePort uint16, upstreams map[string]bool, port corev1.ServicePort) (*maps.NatKey, *maps.NatVal) {
	eipAddr := net.ParseIP(eIP)
	natKey := new(maps.NatKey)
	natKey.Sys = uint32(sysId)
	natKey.Daddr[0], natKey.Daddr[1], natKey.Daddr[2], natKey.Daddr[3], natKey.V6, _ = util.IPToInt(eipAddr)
	natKey.Dport = util.HostToNetShort(ePort)
	natKey.Proto = uint8(maps.IPPROTO_TCP)
	if eipAddr.To4() == nil {
		natKey.V6 = 1
	}

	natVal := new(maps.NatVal)
	for rip, microSvc := range upstreams {
		ripAddr := net.ParseIP(rip)
		if natKey.V6 == 1 && ripAddr.To16() == nil {
			continue
		}
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

func (s *Server) getE4lbPodNat(sysId maps.SysID, eIP string, ePort uint16, upstreams map[string]bool, port corev1.ServicePort) (*maps.NatKey, *maps.NatVal) {
	eipAddr := net.ParseIP(eIP)
	natKey := new(maps.NatKey)
	natKey.Sys = uint32(sysId)
	natKey.Daddr[0], natKey.Daddr[1], natKey.Daddr[2], natKey.Daddr[3], natKey.V6, _ = util.IPToInt(eipAddr)
	natKey.Dport = util.HostToNetShort(ePort)
	natKey.Proto = uint8(maps.IPPROTO_TCP)
	if eipAddr.To4() == nil {
		natKey.V6 = 1
	}

	natVal := new(maps.NatVal)
	for rip, microSvc := range upstreams {
		ripAddr := net.ParseIP(rip)
		var brVal *maps.IFaceVal
		if natKey.V6 == 1 && ripAddr.To16() == nil {
			brVal = s.getCniBridge6Info()
		} else {
			brVal = s.getCniBridge4Info()
		}
		if brVal == nil {
			continue
		}
		if microSvc {
			if port.TargetPort.IntVal > 0 && port.TargetPort.IntVal <= math.MaxUint16 {
				rport := uint16(port.TargetPort.IntVal)
				natVal.AddEp(ripAddr, rport, brVal.Mac[:], 0, maps.BPF_F_EGRESS, nil, true)
			} else {
				continue
			}
		} else {
			rport := ePort
			natVal.AddEp(ripAddr, rport, brVal.Mac[:], 0, maps.BPF_F_EGRESS, nil, true)
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

func (s *Server) setupE4LBPodNat(natKey *maps.NatKey, natVal *maps.NatVal) error {
	return maps.AddNatEntry(maps.SysMesh, natKey, natVal)
}

func (s *Server) unsetE4LBPodNat(natKey *maps.NatKey) error {
	return maps.DelNatEntry(maps.SysMesh, natKey)
}

func (s *Server) setupE4LBServiceNeigh(defaultIfi int, eip string, defaultHwAddr net.HardwareAddr) {
	neigh := &netlink.Neigh{
		LinkIndex:    defaultIfi,
		State:        arp.NUD_REACHABLE,
		IP:           net.ParseIP(eip),
		HardwareAddr: defaultHwAddr,
	}
	if err := netlink.NeighSet(neigh); err != nil {
		log.Error().Msg(err.Error())
		if err = netlink.NeighAdd(neigh); err != nil {
			log.Error().Msg(err.Error())
		}
	}
}

// IsE4LBEnabled checks if the service is enabled for flb
func IsE4LBEnabled(svc *corev1.Service, kubeClient kubernetes.Interface) bool {
	if svc == nil {
		return false
	}

	// if service doesn't have flb.flomesh.io/enabled annotation
	if svc.Annotations == nil || svc.Annotations[constants.FLBEnabledAnnotation] == "" {
		// check ns annotation
		ns, err := kubeClient.CoreV1().
			Namespaces().
			Get(context.TODO(), svc.Namespace, metav1.GetOptions{})

		if err != nil {
			log.Error().Msgf("Failed to get namespace %q: %s", svc.Namespace, err)
			return false
		}

		if ns.Annotations == nil || ns.Annotations[constants.FLBEnabledAnnotation] == "" {
			return false
		}

		log.Debug().Msgf("Found annotation %q on Namespace %q", constants.FLBEnabledAnnotation, ns.Name)
		return utils.ParseEnabled(ns.Annotations[constants.FLBEnabledAnnotation])
	}

	// parse svc annotation
	log.Debug().Msgf("Found annotation %q on Service %s/%s", constants.FLBEnabledAnnotation, svc.Namespace, svc.Name)
	return utils.ParseEnabled(svc.Annotations[constants.FLBEnabledAnnotation])
}
