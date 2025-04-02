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
	defaultIfi, defaultEth, defaultHwAddr, err := s.discoverGateway()
	if err != nil {
		log.Error().Err(err).Msg(`fail to discover gateway`)
		return
	}

	obsoleteNats := make(map[string]*XNat)
	for natKey, natVal := range s.xnatCache {
		if natVal.key.Sys == uint32(maps.SysE4lb) && natVal.key.TcDir == uint8(maps.TC_DIR_IGR) {
			obsoleteNats[natKey] = natVal
		}
	}

	for uid, k8sSvc := range e4lbSvcs {
		eip, exists := e4lbEips[uid]
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
			vport := uint16(0)
			if port.Port > 0 && port.Port <= math.MaxUint16 {
				vport = uint16(port.Port)
			}

			eipAddr := net.ParseIP(eip)
			natKey := s.getE4lbNatKey(eipAddr, vport)
			natVal := s.getE4lbNatVal(upstreams, natKey, port, vport)

			e4lbNat := newXNat(maps.SysE4lb, natKey, natVal)
			if existsNat, exists := s.xnatCache[e4lbNat.Key()]; !exists {
				s.setupE4LBServiceNeigh(defaultIfi, eip, defaultHwAddr)
				if err := s.setupE4LBServiceNat(natKey, natVal); err != nil {
					log.Error().Err(err).Msgf(`failed to setup e4lb nat, eip: %s`, eip)
					continue
				}
				s.xnatCache[e4lbNat.Key()] = e4lbNat
			} else {
				if existsNat.valHash != e4lbNat.valHash {
					s.setupE4LBServiceNeigh(defaultIfi, eip, defaultHwAddr)
					if err := s.setupE4LBServiceNat(natKey, natVal); err != nil {
						log.Error().Err(err).Msgf(`failed to setup e4lb nat, eip: %s`, eip)
						continue
					}
					s.xnatCache[e4lbNat.Key()] = e4lbNat
				}
				delete(obsoleteNats, e4lbNat.Key())
			}
		}

		if err := arp.Announce(defaultEth, eip, defaultHwAddr); err != nil {
			log.Error().Msg(err.Error())
		}
	}

	if len(obsoleteNats) > 0 {
		for natKey, e4lbNat := range obsoleteNats {
			if err := s.unsetE4LBServiceNat(&e4lbNat.key); err != nil {
				log.Error().Err(err).Msgf(`failed to unset e4lb nat`)
				continue
			}
			delete(s.xnatCache, natKey)
		}
	}
}

func (s *Server) getE4lbNatKey(eipAddr net.IP, vport uint16) *maps.NatKey {
	natKey := new(maps.NatKey)
	natKey.Daddr[0], natKey.Daddr[1], natKey.Daddr[2], natKey.Daddr[3], natKey.V6, _ = util.IPToInt(eipAddr)
	natKey.Dport = util.HostToNetShort(vport)
	natKey.Proto = uint8(maps.IPPROTO_TCP)
	if eipAddr.To4() == nil {
		natKey.V6 = 1
	}
	return natKey
}

func (s *Server) getE4lbNatVal(upstreams map[string]bool, natKey *maps.NatKey, port corev1.ServicePort, vport uint16) *maps.NatVal {
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
			rport := vport
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
	return natVal
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

func (s *Server) setupE4LBServiceNat(natKey *maps.NatKey, natVal *maps.NatVal) error {
	for _, tcDir := range []maps.TcDir{maps.TC_DIR_IGR, maps.TC_DIR_EGR} {
		natKey.TcDir = uint8(tcDir)
		if err := maps.AddNatEntry(maps.SysE4lb, natKey, natVal); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) unsetE4LBServiceNat(natKey *maps.NatKey) error {
	for _, tcDir := range []maps.TcDir{maps.TC_DIR_IGR, maps.TC_DIR_EGR} {
		natKey.TcDir = uint8(tcDir)
		if err := maps.DelNatEntry(maps.SysE4lb, natKey); err != nil {
			return err
		}
	}
	return nil
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
