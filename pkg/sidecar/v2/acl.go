package v2

import (
	"net"
	"sync"

	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/maps"
	"github.com/flomesh-io/fsm/pkg/xnetwork/xnet/util"
)

var aclCache map[uint32]uint8
var aclLock sync.Mutex

func (s *Server) updateAcls(aclAddrs map[uint32]uint8) {
	aclLock.Lock()
	defer aclLock.Unlock()

	if aclCache == nil {
		aclCache = make(map[uint32]uint8)
		if aclEntries := maps.GetAclEntries(maps.SysMesh); len(aclEntries) > 0 {
			for aclKey, aclVal := range aclEntries {
				if aclVal.Id == aclId && aclVal.Flag == aclFlag {
					aclCache[aclKey.Addr[0]] = aclVal.Acl
				}
			}
		}
	}

	var deleteKeys []maps.AclKey
	var addKeys []maps.AclKey
	var addVals []maps.AclVal

	for addr := range aclCache {
		if _, exists := aclAddrs[addr]; !exists {
			delKey := maps.AclKey{}
			delKey.Addr[0] = addr
			delKey.Port = util.HostToNetShort(0)
			delKey.Proto = uint8(maps.IPPROTO_TCP)
			deleteKeys = append(deleteKeys, delKey)
		}
	}

	for addr, acl := range aclAddrs {
		if _, exists := aclCache[addr]; !exists {
			addKey := maps.AclKey{}
			addKey.Addr[0] = addr
			addKey.Port = util.HostToNetShort(0)
			addKey.Proto = uint8(maps.IPPROTO_TCP)
			addKeys = append(addKeys, addKey)

			addVal := maps.AclVal{}
			addVal.Flag = aclFlag
			addVal.Id = aclId
			addVal.Acl = acl
			addVals = append(addVals, addVal)
		}
	}

	if len(deleteKeys) > 0 {
		if _, err := maps.DelAclEntries(maps.SysMesh, deleteKeys); err != nil {
			log.Error().Err(err).Msg(`failed to delete acls`)
		} else {
			for _, key := range deleteKeys {
				delete(aclCache, key.Addr[0])
			}
		}
	}

	if len(addKeys) > 0 {
		if _, err := maps.AddAclEntries(maps.SysMesh, addKeys, addVals); err != nil {
			log.Error().Err(err).Msg(`failed to add acls`)
		} else {
			for idx, key := range addKeys {
				aclCache[key.Addr[0]] = addVals[idx].Acl
			}
		}
	}
}

func (s *Server) doConfigAcls() {
	aclAddrs := make(map[uint32]uint8)
	acls := s.xnetworkController.GetAccessControls()
	for _, acl := range acls {
		if len(acl.Spec.Services) > 0 {
			for _, aclSvc := range acl.Spec.Services {
				meshSvc := service.MeshService{Name: aclSvc.Name}
				if len(aclSvc.Namespace) > 0 {
					meshSvc.Namespace = aclSvc.Namespace
				} else {
					meshSvc.Namespace = acl.Namespace
				}
				if k8sSvc := s.kubeController.GetService(meshSvc); k8sSvc != nil {
					if aclSvc.WithClusterIPs {
						clusterIPNb, _ := util.IPv4ToInt(net.ParseIP(k8sSvc.Spec.ClusterIP))
						aclAddrs[clusterIPNb] = uint8(maps.ACL_TRUSTED)
						for _, clusterIP := range k8sSvc.Spec.ClusterIPs {
							clusterIPNb, _ = util.IPv4ToInt(net.ParseIP(clusterIP))
							aclAddrs[clusterIPNb] = uint8(maps.ACL_TRUSTED)
						}
					}

					if aclSvc.WithExternalIPs {
						for _, ingress := range k8sSvc.Status.LoadBalancer.Ingress {
							ingressIPNb, _ := util.IPv4ToInt(net.ParseIP(ingress.IP))
							aclAddrs[ingressIPNb] = uint8(maps.ACL_TRUSTED)
						}
					}

					if aclSvc.WithEndpointIPs {
						if eps, err := s.kubeController.GetEndpoints(meshSvc); err == nil && eps != nil {
							for _, subsets := range eps.Subsets {
								for _, epAddr := range subsets.Addresses {
									epIPNb, _ := util.IPv4ToInt(net.ParseIP(epAddr.IP))
									aclAddrs[epIPNb] = uint8(maps.ACL_TRUSTED)
								}
							}
						}
					}
				}
			}
		}
	}

	s.updateAcls(aclAddrs)
	s.updateDNSNat()
}
