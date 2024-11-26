package v2

import (
	"sync"

	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/maps"
	"github.com/flomesh-io/fsm/pkg/sidecar/v2/xnet/util"
)

var aclCache map[uint32]uint8
var aclLock sync.Mutex

func (s *Server) updateAcls(aclAddrs map[uint32]uint8) {
	aclLock.Lock()
	defer aclLock.Unlock()

	if aclCache == nil {
		aclCache = make(map[uint32]uint8)
		if aclEntries := maps.GetAclEntries(); len(aclEntries) > 0 {
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
		if _, err := maps.DelAclEntries(deleteKeys); err != nil {
			log.Error().Err(err).Msg(`failed to delete acls`)
		} else {
			for _, key := range deleteKeys {
				delete(aclCache, key.Addr[0])
			}
		}
	}

	if len(addKeys) > 0 {
		if _, err := maps.AddAclEntries(addKeys, addVals); err != nil {
			log.Error().Err(err).Msg(`failed to add acls`)
		} else {
			for idx, key := range addKeys {
				aclCache[key.Addr[0]] = addVals[idx].Acl
			}
		}
	}
}
