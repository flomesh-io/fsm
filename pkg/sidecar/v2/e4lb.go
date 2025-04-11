package v2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/strings/slices"

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
	readyNodes, existsE4lbNode := availableNetworkNodes(s.kubeClient)
	if len(readyNodes) == 0 {
		return
	} else if _, exists := readyNodes[s.nodeName]; !exists {
		return
	}

	e4lbSvcs := make(map[types.UID]*corev1.Service)
	e4lbEips := make(map[types.UID]string)
	s.processEIPAdvertisements(readyNodes, existsE4lbNode, e4lbSvcs, e4lbEips)
	s.processServiceAnnotations(readyNodes, existsE4lbNode, e4lbSvcs, e4lbEips)
	s.announceE4LBService(e4lbSvcs, e4lbEips)
}

func (s *Server) processServiceAnnotations(readyNodes map[string]bool, existsE4lbNode bool, e4lbSvcs map[types.UID]*corev1.Service, e4lbEips map[types.UID]string) {
	k8sSvcs := s.kubeController.ListServices(false, true)
	if len(k8sSvcs) > 0 {
		for _, k8sSvc := range k8sSvcs {
			if !IsE4LBEnabled(k8sSvc, s.kubeClient) {
				continue
			}

			eip := k8sSvc.Annotations[constants.FLBDesiredIPAnnotation]
			ipAddr := net.ParseIP(eip)
			if ipAddr == nil || (ipAddr.To4() == nil && ipAddr.To16() == nil) || ipAddr.IsUnspecified() || ipAddr.IsMulticast() {
				continue
			}

			var availableNodes []string
			for nodeName, e4lbEnabled := range readyNodes {
				if existsE4lbNode {
					if e4lbEnabled {
						availableNodes = append(availableNodes, nodeName)
					}
				} else {
					availableNodes = append(availableNodes, nodeName)
				}
			}
			if len(availableNodes) == 0 {
				continue
			}

			sort.Slice(availableNodes, func(i, j int) bool {
				hi := sha256.Sum256([]byte(availableNodes[i] + "#" + eip))
				hj := sha256.Sum256([]byte(availableNodes[j] + "#" + eip))

				return bytes.Compare(hi[:], hj[:]) < 0
			})

			if availableNodes[0] == s.nodeName {
				e4lbSvcs[k8sSvc.GetUID()] = k8sSvc
				e4lbEips[k8sSvc.GetUID()] = eip
			}
		}
	}
}

func (s *Server) processEIPAdvertisements(readyNodes map[string]bool, existsE4lbNode bool, e4lbSvcs map[types.UID]*corev1.Service, e4lbEips map[types.UID]string) {
	eipAdvs := s.xnetworkController.GetEIPAdvertisements()
	if len(eipAdvs) > 0 {
		for _, eipAdv := range eipAdvs {
			var availableNodes []string
			if len(eipAdv.Spec.Nodes) > 0 {
				if !slices.Contains(eipAdv.Spec.Nodes, s.nodeName) {
					continue
				}
				for _, nodeName := range eipAdv.Spec.Nodes {
					if _, exists := readyNodes[nodeName]; exists {
						availableNodes = append(availableNodes, nodeName)
					}
				}
			} else {
				for nodeName, e4lbEnabled := range readyNodes {
					if existsE4lbNode {
						if e4lbEnabled {
							availableNodes = append(availableNodes, nodeName)
						}
					} else {
						availableNodes = append(availableNodes, nodeName)
					}
				}
			}
			if len(availableNodes) == 0 {
				continue
			}

			meshSvc := service.MeshService{Name: eipAdv.Spec.Service.Name}
			if len(eipAdv.Spec.Service.Namespace) > 0 {
				meshSvc.Namespace = eipAdv.Spec.Service.Namespace
			} else {
				meshSvc.Namespace = eipAdv.Namespace
			}
			k8sSvc := s.kubeController.GetService(meshSvc)
			if k8sSvc == nil {
				continue
			}

			eip := eipAdv.Spec.EIP
			ipAddr := net.ParseIP(eip)
			if ipAddr == nil || (ipAddr.To4() == nil && ipAddr.To16() == nil) || ipAddr.IsUnspecified() || ipAddr.IsMulticast() {
				continue
			}

			sort.Slice(availableNodes, func(i, j int) bool {
				hi := sha256.Sum256([]byte(availableNodes[i] + "#" + eip))
				hj := sha256.Sum256([]byte(availableNodes[j] + "#" + eip))

				return bytes.Compare(hi[:], hj[:]) < 0
			})

			if availableNodes[0] == s.nodeName {
				e4lbSvcs[k8sSvc.GetUID()] = k8sSvc
				e4lbEips[k8sSvc.GetUID()] = eip
			}
		}
	}
}

func (s *Server) announceE4LBService(e4lbSvcs map[types.UID]*corev1.Service, e4lbEips map[types.UID]string) {
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

func (s *Server) setupE4LBNats(e4lbSvcs map[types.UID]*corev1.Service, e4lbEips map[types.UID]string, obsoleteNats map[string]*XNat) {
	defaultIfi, defaultEth, defaultHwAddr, err := s.discoverGateway()
	if err != nil {
		log.Error().Err(err).Msg(`fail to discover gateway`)
		return
	}

	for uid, k8sSvc := range e4lbSvcs {
		eIP, exists := e4lbEips[uid]
		if !exists {
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

		for _, port := range k8sSvc.Spec.Ports {
			if !strings.EqualFold(string(port.Protocol), string(corev1.ProtocolTCP)) {
				continue
			}
			ePort := uint16(0)
			if port.Port > 0 && port.Port <= math.MaxUint16 {
				ePort = uint16(port.Port)
			}

			s.setupE4LBServiceNeigh(defaultIfi, eIP, defaultHwAddr)

			nodeNatKey, nodeNatVal := s.getE4lbNodeNat(maps.SysE4lb, eIP, ePort, upstreams, port)
			for _, tcDir := range []maps.TcDir{maps.TC_DIR_IGR, maps.TC_DIR_EGR} {
				nodeNatKey.TcDir = uint8(tcDir)
				nodeNat := newXNat(nodeNatKey, nodeNatVal)
				log.Debug().Msgf("nodeNat.Key: %s", nodeNat.Key())
				if existsNat, exists := s.xnatCache[nodeNat.Key()]; !exists {
					if err := s.setupE4LBNodeNat(nodeNatKey, nodeNatVal); err == nil {
						log.Debug().Msgf("nodeNat.Key() cache add: %s", nodeNat.Key())
						s.xnatCache[nodeNat.Key()] = nodeNat
					} else {
						log.Error().Err(err).Msgf(`failed to setup e4lb node nat, eip: %s`, eIP)
					}
				} else {
					if existsNat.valHash != nodeNat.valHash {
						if err := s.setupE4LBNodeNat(nodeNatKey, nodeNatVal); err == nil {
							log.Debug().Msgf("nodeNat.Key() cache add: %s", nodeNat.Key())
							s.xnatCache[nodeNat.Key()] = nodeNat
						} else {
							log.Error().Err(err).Msgf(`failed to setup e4lb node nat, eip: %s`, eIP)
						}
					}
					log.Debug().Msgf("nodeNat.Key() obsoleteNats del: %s", nodeNat.Key())
					delete(obsoleteNats, nodeNat.Key())
				}
			}

			podNatKey, podNatVal := s.getE4lbPodNat(maps.SysMesh, eIP, ePort, upstreams, port)
			for _, tcDir := range []maps.TcDir{maps.TC_DIR_EGR} {
				podNatKey.TcDir = uint8(tcDir)
				podNat := newXNat(podNatKey, podNatVal)
				log.Debug().Msgf("podNat.Key: %s", podNat.Key())
				if existsNat, exists := s.xnatCache[podNat.Key()]; !exists {
					if err := s.setupE4LBPodNat(podNatKey, podNatVal); err == nil {
						log.Debug().Msgf("podNat.Key() cache add: %s", podNat.Key())
						s.xnatCache[podNat.Key()] = podNat
					} else {
						log.Error().Err(err).Msgf(`failed to setup e4lb pod nat, eip: %s`, eIP)
					}
				} else {
					if existsNat.valHash != podNat.valHash {
						if err := s.setupE4LBPodNat(podNatKey, podNatVal); err == nil {
							log.Debug().Msgf("podNat.Key() cache add: %s", podNat.Key())
							s.xnatCache[podNat.Key()] = podNat
						} else {
							log.Error().Err(err).Msgf(`failed to setup e4lb pod nat, eip: %s`, eIP)
						}
					}
					log.Debug().Msgf("podNat.Key() obsoleteNats del: %s", podNat.Key())
					delete(obsoleteNats, podNat.Key())
				}
			}
		}

		if err := arp.Announce(defaultEth, eIP, defaultHwAddr); err != nil {
			log.Error().Msg(err.Error())
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
