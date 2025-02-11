package v2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/strings/slices"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/arp"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/maps"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/route"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/util"
)

var (
	bridgeInfo *maps.IFaceVal
)

type E4LBNat struct {
	vip   string
	vport uint16
	rip   string
	rport uint16
}

func (lb *E4LBNat) Key() string {
	return fmt.Sprintf("%s:%d:%s:%d", lb.vip, lb.vport, lb.rip, lb.rport)
}

// intToIPv4 converts IP address of version 4 from integer to net.IP
// representation.
func intToIPv4(ipaddr uint32) net.IP {
	ip := make(net.IP, net.IPv4len)
	// Proceed conversion
	binary.LittleEndian.PutUint32(ip, ipaddr)
	return ip
}

// netToHostShort converts a 16-bit integer from network to host byte order, aka "ntohs"
func netToHostShort(i uint16) uint16 {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, i)
	return binary.LittleEndian.Uint16(data)
}

func (s *Server) loadNatEntries() error {
	natEntries, err := maps.ListNatEntries()
	if err == nil {
		for natKey, natVal := range natEntries {
			if natKey.Sys == uint32(maps.SysE4lb) && natKey.TcDir == uint8(maps.TC_DIR_IGR) {
				for n := uint16(0); n < natVal.EpCnt; n++ {
					if natVal.Eps[n].Active > 0 {
						e4lbNat := E4LBNat{
							vip:   intToIPv4(natKey.Daddr[0]).String(),
							vport: netToHostShort(natKey.Dport),
							rip:   intToIPv4(natVal.Eps[n].Raddr[0]).String(),
							rport: netToHostShort(natVal.Eps[n].Rport),
						}
						s.e4lbNatCache[e4lbNat.Key()] = &e4lbNat
					}
				}
			}
		}
	}
	return err
}

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
			if ipAddr == nil || ipAddr.To4() == nil || ipAddr.IsUnspecified() || ipAddr.IsMulticast() {
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
			if ipAddr == nil || ipAddr.To4() == nil || ipAddr.IsUnspecified() || ipAddr.IsMulticast() {
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
	var defaultEth string
	var defaultHwAddr net.HardwareAddr
	if dev, _, err := route.DiscoverGateway(); err != nil {
		log.Error().Msg(err.Error())
		return
	} else if viaEth, err := netlink.LinkByName(dev); err != nil {
		log.Error().Msg(err.Error())
		return
	} else {
		defaultHwAddr = viaEth.Attrs().HardwareAddr
		defaultEth = dev
	}

	obsoleteNats := make(map[string]*E4LBNat)
	for natKey, natVal := range s.e4lbNatCache {
		obsoleteNats[natKey] = natVal
	}

	for uid, k8sSvc := range e4lbSvcs {
		eip, exists := e4lbEips[uid]
		if !exists {
			continue
		}
		if len(k8sSvc.Spec.ClusterIP) == 0 {
			continue
		}

		var ports []uint16
		for _, port := range k8sSvc.Spec.Ports {
			if !strings.EqualFold(string(port.Protocol), string(corev1.ProtocolTCP)) {
				continue
			}
			if port.Port > 0 && port.Port <= math.MaxUint16 {
				ports = append(ports, uint16(port.Port))
			}
		}
		if len(ports) == 0 {
			continue
		}

		for _, port := range ports {
			nat := E4LBNat{
				vip:   eip,
				vport: port,
				rip:   k8sSvc.Spec.ClusterIP,
				rport: port,
			}
			if _, exists = s.e4lbNatCache[nat.Key()]; !exists {
				if err := s.setupE4LBServiceNat(nat.vip, nat.vport, nat.rip, nat.rport); err != nil {
					log.Error().Msg(err.Error())
					continue
				}
				s.e4lbNatCache[nat.Key()] = &nat
			} else {
				delete(obsoleteNats, nat.Key())
			}
		}

		if err := arp.Announce(defaultEth, eip, defaultHwAddr); err != nil {
			log.Error().Msg(err.Error())
		}
	}

	if len(obsoleteNats) > 0 {
		for natKey, natVal := range obsoleteNats {
			if err := s.unsetE4LBServiceNat(natVal.vip, natVal.vport); err != nil {
				log.Error().Msg(err.Error())
				continue
			}
			delete(s.e4lbNatCache, natKey)
		}
	}
}

func (s *Server) setupE4LBServiceNat(vip string, vport uint16, rip string, rport uint16) error {
	natKey := new(maps.NatKey)
	natKey.Daddr[0], natKey.Daddr[1], natKey.Daddr[2], natKey.Daddr[3], natKey.V6, _ = util.IPToInt(net.ParseIP(vip))
	natKey.Dport = util.HostToNetShort(vport)
	natKey.Proto = uint8(maps.IPPROTO_TCP)
	natVal := new(maps.NatVal)
	brVal := s.getBridgeInfo()
	natVal.AddEp(net.ParseIP(rip), rport, brVal.Mac[:], brVal.Ifi, maps.BPF_F_EGRESS, nil, true)
	for _, tcDir := range []maps.TcDir{maps.TC_DIR_IGR, maps.TC_DIR_EGR} {
		natKey.TcDir = uint8(tcDir)
		if err := maps.AddNatEntry(maps.SysE4lb, natKey, natVal); err != nil {
			return fmt.Errorf(`failed to setup e4lb nat, vip: %s`, vip)
		}
	}
	return nil
}

func (s *Server) unsetE4LBServiceNat(vip string, vport uint16) error {
	natKey := new(maps.NatKey)
	natKey.Daddr[0], natKey.Daddr[1], natKey.Daddr[2], natKey.Daddr[3], natKey.V6, _ = util.IPToInt(net.ParseIP(vip))
	natKey.Dport = util.HostToNetShort(vport)
	natKey.Proto = uint8(maps.IPPROTO_TCP)
	for _, tcDir := range []maps.TcDir{maps.TC_DIR_IGR, maps.TC_DIR_EGR} {
		natKey.TcDir = uint8(tcDir)
		if err := maps.DelNatEntry(maps.SysE4lb, natKey); err != nil {
			return fmt.Errorf(`failed to unset e4lb nat, vip: %s`, vip)
		}
	}
	return nil
}

func (s *Server) getBridgeInfo() *maps.IFaceVal {
	if bridgeInfo != nil {
		return bridgeInfo
	}

	brKey := new(maps.IFaceKey)
	brKey.Len = uint8(len(bridgeDev))
	copy(brKey.Name[0:brKey.Len], bridgeDev)
	for {
		var err error
		bridgeInfo, err = maps.GetIFaceEntry(brKey)
		if err != nil {
			log.Error().Err(err).Msg(`failed to get node bridge info`)
			time.Sleep(time.Second * 5)
			continue
		}
		if bridgeInfo == nil {
			log.Error().Msg(`failed to get node bridge info`)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}
	return bridgeInfo
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
