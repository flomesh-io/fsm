package v2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"sort"

	mapset "github.com/deckarep/golang-set"
	"github.com/mitchellh/hashstructure/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xnetv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/xnetwork/v1alpha1"
	xnetworkClientset "github.com/flomesh-io/fsm/pkg/gen/client/xnetwork/clientset/versioned"
)

func (topo *e4lbTopo) loadEIPAdvertisements(eipAdvs []*xnetv1alpha1.EIPAdvertisement) {
	if len(eipAdvs) > 0 {
		for _, eipAdv := range eipAdvs {
			if len(eipAdv.Spec.EIPs) > 0 {
				for _, eip := range eipAdv.Spec.EIPs {
					topo.eipSvcCache[eip] = 0
				}
			}
		}
		for _, eipAdv := range eipAdvs {
			hash, _ := hashstructure.Hash(eipAdv.Status.Announce, hashstructure.FormatV2,
				&hashstructure.HashOptions{
					ZeroNil:         true,
					IgnoreZeroValue: true,
					SlicesAsSets:    true,
				})
			topo.advertisementHash[eipAdv.UID] = hash
			if len(eipAdv.Status.Announce) > 0 {
				for eip, node := range eipAdv.Status.Announce {
					if len(node) > 0 {
						if eipSet, existsNode := topo.nodeEipLayout[node]; existsNode {
							if eipSvc, existsEipSvc := topo.eipSvcCache[eip]; existsEipSvc {
								if _, existsEip := eipSet[eip]; !existsEip {
									eipSet[eip] = eipSvc
								}
								topo.eipNodeLayout[eip] = node
							}
						}
					}
				}
			}
		}
	}
}

func (topo *e4lbTopo) processEIPAdvertisements(eipAdvs []*xnetv1alpha1.EIPAdvertisement, xnetworkClient xnetworkClientset.Interface) {
	for _, eipAdv := range eipAdvs {
		statusAnnounce := make(map[string]string)
		availableNodeSet := mapset.NewSet()
		topo.fillAvailableNodeSet(eipAdv, availableNodeSet)
		if availableNodeSet.Cardinality() == 0 {
			continue
		}

		for _, eip := range eipAdv.Spec.EIPs {
			if selectedNode, assigned := topo.eipNodeLayout[eip]; assigned {
				availableNodeSet.Remove(selectedNode)
			}
		}

		for _, eip := range eipAdv.Spec.EIPs {
			if selectedNode, assigned := topo.eipNodeLayout[eip]; assigned {
				statusAnnounce[eip] = selectedNode
				continue
			}

			if availableNodeSet.Cardinality() == 0 {
				topo.fillAvailableNodeSet(eipAdv, availableNodeSet)
			}

			var selectedNode string
			if availableNodeSet.Cardinality() > 0 {
				availableNodes := availableNodeSet.ToSlice()
				if len(availableNodes) > 1 {
					sort.Slice(availableNodes, func(i, j int) bool {
						ci := 0
						cj := 0
						ni := availableNodes[i].(string)
						nj := availableNodes[j].(string)
						if eipSet, exists := topo.nodeEipLayout[ni]; exists {
							ci = len(eipSet)
						}
						if eipSet, exists := topo.nodeEipLayout[nj]; exists {
							cj = len(eipSet)
						}
						if ci == cj {
							hi := sha256.Sum256([]byte(ni))
							hj := sha256.Sum256([]byte(nj))
							return bytes.Compare(hi[:], hj[:]) < 0
						}
						return ci < cj
					})
				}
				selectedNode = availableNodes[0].(string)
			}

			topo.eipNodeLayout[eip] = selectedNode
			statusAnnounce[eip] = selectedNode
			if len(selectedNode) > 0 {
				topo.nodeEipLayout[selectedNode][eip] = 0
				availableNodeSet.Remove(selectedNode)
			}
		}

		curHash, _ := hashstructure.Hash(statusAnnounce, hashstructure.FormatV2,
			&hashstructure.HashOptions{
				ZeroNil:         true,
				IgnoreZeroValue: true,
				SlicesAsSets:    true,
			})
		preHash := topo.advertisementHash[eipAdv.UID]
		if curHash != preHash {
			preAnnounce := eipAdv.Status.Announce
			eipAdv.Status.Announce = statusAnnounce
			if _, err := xnetworkClient.XnetworkV1alpha1().EIPAdvertisements(eipAdv.Namespace).
				UpdateStatus(context.TODO(), eipAdv, metav1.UpdateOptions{}); err != nil {
				eipAdv.Status.Announce = preAnnounce
				log.Error().Err(err).Msgf("fail to update status for EIPAdvertisement: %s/%s", eipAdv.Namespace, eipAdv.Name)
			} else {
				topo.advertisementHash[eipAdv.UID] = curHash
			}
		}
	}
}

func (topo *e4lbTopo) fillAvailableNodeSet(eipAdv *xnetv1alpha1.EIPAdvertisement, availableNodeSet mapset.Set) {
	if len(eipAdv.Spec.Nodes) > 0 {
		for _, nodeName := range eipAdv.Spec.Nodes {
			if _, exists := topo.nodeCache[nodeName]; exists {
				availableNodeSet.Add(nodeName)
			}
		}
	} else {
		for nodeName, e4lbEnabled := range topo.nodeCache {
			if topo.existsE4lbNodes {
				if e4lbEnabled {
					availableNodeSet.Add(nodeName)
				}
			} else {
				availableNodeSet.Add(nodeName)
			}
		}
	}
}
