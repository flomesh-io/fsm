// Package registry implements handler's methods.
package registry

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/announcements"
	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	sidecarv1 "github.com/flomesh-io/fsm/pkg/sidecar/v1"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
)

// ReleaseCertificateHandler releases certificates based on podDelete events
func (pr *ProxyRegistry) ReleaseCertificateHandler(certManager certificateReleaser, stop <-chan struct{}) {
	kubePubSub := pr.msgBroker.GetKubeEventPubSub()
	deleteChan := kubePubSub.Sub(announcements.PodDeleted.String(), announcements.VirtualMachineDeleted.String())
	defer pr.msgBroker.Unsub(kubePubSub, deleteChan)

	for {
		select {
		case <-stop:
			return

		case deletedMsg := <-deleteChan:
			psubMessage, castOk := deletedMsg.(events.PubSubMessage)
			if !castOk {
				log.Error().Msgf("Error casting to events.PubSubMessage, got type %T", psubMessage)
				continue
			}

			// guaranteed can only be a PodDeleted event
			deletedPodObj, podCastOk := psubMessage.OldObj.(*corev1.Pod)
			if podCastOk {
				proxyUUID := deletedPodObj.Labels[constants.SidecarUniqueIDLabelName]
				if proxyIface, ok := connectedProxies.Load(proxyUUID); ok {
					proxy := proxyIface.(*pipy.Proxy)
					log.Info().Msgf("Pod with label %s: %s found in proxy registry; releasing certificate for proxy %s", constants.SidecarUniqueIDLabelName, proxyUUID, proxy.Identity)
					certManager.ReleaseCertificate(sidecarv1.NewCertCNPrefix(proxy.UUID, proxy.Kind(), proxy.Identity))
					if pr.UpdateProxies != nil {
						pr.UpdateProxies()
					}
				} else {
					log.Info().Msgf("Pod with label %s: %s not found in proxy registry", constants.SidecarUniqueIDLabelName, proxyUUID)
				}
				continue
			}

			// guaranteed can only be a VirtualMachineDeleted event
			deletedVmObj, vmCastOk := psubMessage.OldObj.(*machinev1alpha1.VirtualMachine)
			if vmCastOk {
				proxyUUID := deletedVmObj.Labels[constants.SidecarUniqueIDLabelName]
				if proxyIface, ok := connectedProxies.Load(proxyUUID); ok {
					proxy := proxyIface.(*pipy.Proxy)
					log.Info().Msgf("VM with label %s: %s found in proxy registry; releasing certificate for proxy %s", constants.SidecarUniqueIDLabelName, proxyUUID, proxy.Identity)
					certManager.ReleaseCertificate(sidecarv1.NewCertCNPrefix(proxy.UUID, proxy.Kind(), proxy.Identity))
					if pr.UpdateProxies != nil {
						pr.UpdateProxies()
					}
				} else {
					log.Info().Msgf("VM with label %s: %s not found in proxy registry", constants.SidecarUniqueIDLabelName, proxyUUID)
				}
				continue
			}
		}
	}
}
