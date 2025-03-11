// Package repo implements broadcast's methods.
package repo

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/announcements"
	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
	"github.com/flomesh-io/fsm/pkg/models"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/registry"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

// Routine which fulfills listening to proxy broadcasts
func (s *Server) broadcastListener() {
	// Register for proxy config updates broadcast by the message broker
	proxyUpdatePubSub := s.msgBroker.GetProxyUpdatePubSub()
	proxyUpdateChan := proxyUpdatePubSub.Sub(announcements.ProxyUpdate.String())
	defer s.msgBroker.Unsub(proxyUpdatePubSub, proxyUpdateChan)

	proxyCreationChan := s.msgBroker.GetProxyCreationChan()
	proxyDeletionChan := s.msgBroker.GetProxyDeletionChan()
	proxyWorkQueues := workerpool.NewWorkerPool(64)

	timerDuration := time.Second * 20

	slidingTimer := time.NewTimer(timerDuration * 2)
	defer slidingTimer.Stop()

	slidingTimerReset := func() {
		slidingTimer.Reset(time.Second * 10)
	}

	s.retryProxiesJob = slidingTimerReset
	s.proxyRegistry.UpdateProxies = slidingTimerReset

	reconfirm := false
	pending := false
	blockChan := true

	timerLock := new(sync.Mutex)

	for {
		select {
		case creationPod, ok := <-proxyCreationChan:
			if ok {
				s.fireNewConnectProxy(creationPod, proxyWorkQueues)
			}
		case deletionPod, ok := <-proxyDeletionChan:
			if ok {
				s.fireTermConnectProxy(deletionPod)
			}
		case <-proxyUpdateChan:
			timerLock.Lock()
			if !reconfirm {
				// Wait for an informer synchronization period
				slidingTimer.Reset(timerDuration)
				// Avoid data omission
				reconfirm = true
			} else if !pending {
				pending = true
			}
			timerLock.Unlock()
			blockChan = false
			timerDuration = time.Second * 10
		case <-slidingTimer.C:
			disconnectedProxies, missing := s.fireConnectedProxies()

			timerLock.Lock()
			if reconfirm || pending || blockChan || missing {
				reconfirm = false
				pending = false
				slidingTimer.Reset(timerDuration)
			}
			timerLock.Unlock()

			go func() {
				if len(disconnectedProxies) > 0 {
					for _, proxy := range disconnectedProxies {
						s.proxyRegistry.UnregisterProxy(proxy)
						if _, err := s.repoClient.Delete(fmt.Sprintf("%s/%s", fsmSidecarCodebase, proxy.GetCNPrefix())); err != nil {
							log.Debug().Msgf("fail to delete %s/%s", fsmSidecarCodebase, proxy.GetCNPrefix())
						}
					}
				}
			}()
		}
	}
}

func (s *Server) fireNewConnectProxy(pod *corev1.Pod, proxyNewWorkQueues *workerpool.WorkerPool) {
	if proxy, err := s.fireExistProxy(pod); err == nil {
		if proxy.Metadata == nil || proxy.Addr == nil {
			_ = s.recordPodMetadata(proxy, pod)
		}
		if backlogs := atomic.LoadInt32(&proxy.Backlogs); backlogs > 0 {
			return
		}
		newJob := func() *PipyConfGeneratorJob {
			return &PipyConfGeneratorJob{
				proxy:      proxy,
				repoServer: s,
				done:       make(chan struct{}),
			}
		}
		<-proxyNewWorkQueues.AddJob(newJob())
	}
}

func (s *Server) fireTermConnectProxy(pod *corev1.Pod) {
	if uuid, exists := pod.Labels[constants.SidecarUniqueIDLabelName]; exists {
		s.proxyRegistry.MarkDeletionProxy(uuid)
	}
}

func (s *Server) fireConnectedProxies() (map[string]*pipy.Proxy, bool) {
	connectedProxies := make(map[string]*pipy.Proxy)
	disconnectedProxies := make(map[string]*pipy.Proxy)
	proxies := s.fireExistProxies()
	metricsstore.DefaultMetricsStore.ProxyConnectCount.Set(float64(len(proxies)))
	missing := false
	for _, proxy := range proxies {
		if proxy.Metadata == nil {
			if proxy.VM {
				if err := s.recordVmMetadata(proxy); err != nil {
					missing = true
					continue
				}
			} else {
				if err := s.recordPodMetadata(proxy, nil); err != nil {
					missing = true
					continue
				}
			}
		}
		if proxy.Metadata == nil || proxy.Addr == nil || len(proxy.GetAddr()) == 0 {
			missing = true
			continue
		}
		connectedProxies[proxy.UUID.String()] = proxy
	}

	s.proxyRegistry.RangeConnectedProxy(func(key, value interface{}) bool {
		proxyUUID := key.(string)
		if _, exists := connectedProxies[proxyUUID]; !exists {
			disconnectedProxies[proxyUUID] = value.(*pipy.Proxy)
		}
		return true
	})

	if len(connectedProxies) > 0 {
		for _, proxy := range connectedProxies {
			if backlogs := atomic.LoadInt32(&proxy.Backlogs); backlogs > 0 {
				continue
			}
			newJob := func() *PipyConfGeneratorJob {
				return &PipyConfGeneratorJob{
					proxy:      proxy,
					repoServer: s,
					done:       make(chan struct{}),
				}
			}
			<-s.workQueues.AddJob(newJob())
		}
	}
	return disconnectedProxies, missing
}

func (s *Server) fireExistProxies() []*pipy.Proxy {
	var allProxies []*pipy.Proxy
	allPods := s.kubeController.ListPods()
	for _, pod := range allPods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		proxy, err := GetProxyFromPod(pod)
		if err != nil {
			continue
		}
		proxy = s.fireUpdatedProxy(s.proxyRegistry, proxy)
		if !proxy.Deletion {
			allProxies = append(allProxies, proxy)
		}
	}
	allVms := s.kubeController.ListVms()
	for _, vm := range allVms {
		if vm.DeletionTimestamp != nil {
			continue
		}
		proxy, err := GetProxyFromVm(vm)
		if err != nil {
			continue
		}
		proxy = s.fireUpdatedProxy(s.proxyRegistry, proxy)
		if !proxy.Deletion {
			allProxies = append(allProxies, proxy)
		}
	}
	return allProxies
}

func (s *Server) fireExistProxy(pod *corev1.Pod) (*pipy.Proxy, error) {
	return GetProxyFromPod(pod)
}

func (s *Server) fireUpdatedProxy(proxyRegistry *registry.ProxyRegistry, proxy *pipy.Proxy) *pipy.Proxy {
	connectedProxy := proxyRegistry.GetConnectedProxy(proxy.UUID.String())
	if connectedProxy == nil {
		proxyPtr := &proxy
		callback := func(storedProxyPtr **pipy.Proxy) {
			proxyPtr = storedProxyPtr
		}
		s.informProxy(proxyPtr, callback)
		return *proxyPtr
	}
	return connectedProxy
}

func (s *Server) informProxy(proxyPtr **pipy.Proxy, callback func(**pipy.Proxy)) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if aggregatedErr := s.informTrafficPolicies(proxyPtr, &wg, callback); aggregatedErr != nil {
			log.Error().Err(aggregatedErr).Msgf("Pipy Aggregated Traffic Policies Error.")
		}
	}()
	wg.Wait()
}

// GetProxyFromPod infers and creates a Proxy data structure from a Pod.
// This is a temporary workaround as proxy is required and expected in any vertical call,
// however snapshotcache has no need to provide visibility on proxies whatsoever.
// All verticals use the proxy structure to infer the pod later, so the actual only mandatory
// data for the verticals to be functional is the common name, which links proxy <-> pod
func GetProxyFromPod(pod *corev1.Pod) (*pipy.Proxy, error) {
	uuidString, uuidFound := pod.Labels[constants.SidecarUniqueIDLabelName]
	if !uuidFound {
		return nil, fmt.Errorf("UUID not found for pod %s/%s, not a mesh pod", pod.Namespace, pod.Name)
	}
	proxyUUID, err := uuid.Parse(uuidString)
	if err != nil {
		return nil, fmt.Errorf("Could not parse UUID label into UUID type (%s): %w", uuidString, err)
	}

	return pipy.NewProxy(models.KindSidecar,
		proxyUUID,
		pod.Name,
		pod.Namespace,
		identity.New(pod.Spec.ServiceAccountName, pod.Namespace),
		false,
		nil), nil
}

// GetProxyFromVm infers and creates a Proxy data structure from a VM.
// This is a temporary workaround as proxy is required and expected in any vertical call,
// however snapshotcache has no need to provide visibility on proxies whatsoever.
// All verticals use the proxy structure to infer the VM later, so the actual only mandatory
// data for the verticals to be functional is the common name, which links proxy <-> VM
func GetProxyFromVm(vm *machinev1alpha1.VirtualMachine) (*pipy.Proxy, error) {
	uuidString, uuidFound := vm.Labels[constants.SidecarUniqueIDLabelName]
	if !uuidFound {
		return nil, fmt.Errorf("UUID not found for VM %s/%s, not a mesh VM", vm.Namespace, vm.Name)
	}
	proxyUUID, err := uuid.Parse(uuidString)
	if err != nil {
		return nil, fmt.Errorf("Could not parse UUID label into UUID type (%s): %w", uuidString, err)
	}

	return pipy.NewProxy(models.KindSidecar,
		proxyUUID,
		vm.Name,
		vm.Namespace,
		identity.New(vm.Spec.ServiceAccountName, vm.Namespace),
		true,
		nil), nil
}

// GetProxyUUIDFromPod infers and creates a Proxy UUID from a Pod.
func GetProxyUUIDFromPod(pod *corev1.Pod) (string, error) {
	uuidString, uuidFound := pod.Labels[constants.SidecarUniqueIDLabelName]
	if !uuidFound {
		return "", fmt.Errorf("UUID not found for pod %s/%s, not a mesh pod", pod.Namespace, pod.Name)
	}
	proxyUUID, err := uuid.Parse(uuidString)
	if err != nil {
		return "", fmt.Errorf("Could not parse UUID label into UUID type (%s): %w", uuidString, err)
	}
	return proxyUUID.String(), nil
}

// GetProxyUUIDFromVm infers and creates a Proxy UUID from a VM.
func GetProxyUUIDFromVm(vm *machinev1alpha1.VirtualMachine) (string, error) {
	uuidString, uuidFound := vm.Labels[constants.SidecarUniqueIDLabelName]
	if !uuidFound {
		return "", fmt.Errorf("UUID not found for VM %s/%s, not a mesh VM", vm.Namespace, vm.Name)
	}
	proxyUUID, err := uuid.Parse(uuidString)
	if err != nil {
		return "", fmt.Errorf("Could not parse UUID label into UUID type (%s): %w", uuidString, err)
	}
	return proxyUUID.String(), nil
}
