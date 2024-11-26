package repo

import (
	"strings"
	"sync"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/messaging"
	pipy2 "github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
)

func (s *Server) informTrafficPolicies(proxyPtr **pipy2.Proxy, wg *sync.WaitGroup, callback func(**pipy2.Proxy)) error {
	// If maxDataPlaneConnections is enabled i.e. not 0, then check that the number of Sidecar connections is less than maxDataPlaneConnections
	if s.cfg.GetMaxDataPlaneConnections() != 0 && s.proxyRegistry.GetConnectedProxyCount() >= s.cfg.GetMaxDataPlaneConnections() {
		return errTooManyConnections
	}

	proxy := *proxyPtr
	if proxy.VM {
		if initError := s.recordVmMetadata(proxy); initError == errServiceAccountMismatch {
			// Service Account mismatch
			log.Error().Err(initError).Str("proxy", proxy.String()).Msg("Mismatched service account for proxy")
			return initError
		}
	} else {
		if initError := s.recordPodMetadata(proxy); initError == errServiceAccountMismatch {
			// Service Account mismatch
			log.Error().Err(initError).Str("proxy", proxy.String()).Msg("Mismatched service account for proxy")
			return initError
		}
	}

	proxy = s.proxyRegistry.RegisterProxy(proxy)
	if callback != nil {
		callback(&proxy)
	}

	defer s.proxyRegistry.UnregisterProxy(proxy)

	proxy.Quit = make(chan bool)
	// Subscribe to both broadcast and proxy UUID specific events
	proxyUpdatePubSub := s.msgBroker.GetProxyUpdatePubSub()
	proxyUpdateChan := proxyUpdatePubSub.Sub(announcements.ProxyUpdate.String(), messaging.GetPubSubTopicForProxyUUID(proxy.UUID.String()))
	defer s.msgBroker.Unsub(proxyUpdatePubSub, proxyUpdateChan)

	// Register for certificate rotation updates
	certPubSub := s.msgBroker.GetCertPubSub()
	certRotateChan := certPubSub.Sub(announcements.CertificateRotated.String())
	defer s.msgBroker.Unsub(certPubSub, certRotateChan)

	newJob := func() *PipyConfGeneratorJob {
		return &PipyConfGeneratorJob{
			proxy:      proxy,
			repoServer: s,
			done:       make(chan struct{}),
		}
	}

	wg.Done()

	for {
		select {
		case <-proxy.Quit:
			log.Info().Str("proxy", proxy.String()).Msgf("Pipy Restful session closed")
			return nil

		case <-proxyUpdateChan:
			log.Info().Str("proxy", proxy.String()).Msg("Broadcast update received")
			// Queue a full configuration update
			// Do not send SDS, let sidecar figure out what certs does it want.
			<-s.workQueues.AddJob(newJob())

		case certRotateMsg := <-certRotateChan:
			cert := certRotateMsg.(events.PubSubMessage).NewObj.(*certificate.Certificate)
			if isCNforProxy(proxy, cert.GetCommonName()) {
				// The CN whose corresponding certificate was updated (rotated) by the certificate provider is associated
				// with this proxy, so update the secrets corresponding to this certificate via SDS.
				log.Debug().Str("proxy", proxy.String()).Msg("Certificate has been updated for proxy")

				// Empty DiscoveryRequest should create the SDS specific request
				// Prepare to queue the SDS proxy response job on the worker pool
				<-s.workQueues.AddJob(newJob())
			}
		}
	}
}

// isCNforProxy returns true if the given CN for the workload certificate matches the given proxy's identity.
// Proxy identity corresponds to the k8s service account, while the workload certificate is of the form
// <svc-account>.<namespace>.<trust-domain>.
func isCNforProxy(proxy *pipy2.Proxy, cn certificate.CommonName) bool {
	// Workload certificate CN is of the form <svc-account>.<namespace>.<trust-domain>
	chunks := strings.Split(cn.String(), constants.DomainDelimiter)
	if len(chunks) < 3 {
		return false
	}

	identityForCN := identity.K8sServiceAccount{Name: chunks[0], Namespace: chunks[1]}
	return identityForCN == proxy.Identity.ToK8sServiceAccount()
}

// recordPodMetadata records pod metadata and verifies the certificate issued for this pod
// is for the same service account as seen on the pod's service account
func (s *Server) recordPodMetadata(p *pipy2.Proxy) error {
	if p.Metadata == nil {
		pod, err := s.kubeController.GetPodForProxy(p)
		if err != nil {
			log.Warn().Str("proxy", p.String()).Msg("Could not find pod for connecting proxy. No metadata was recorded.")
			return nil
		}

		workloadKind := ""
		workloadName := ""
		for _, ref := range pod.GetOwnerReferences() {
			if ref.Controller != nil && *ref.Controller {
				workloadKind = ref.Kind
				workloadName = ref.Name
				break
			}
		}

		p.Metadata = &pipy2.ProxyMetadata{
			UID:       string(pod.UID),
			Name:      pod.Name,
			Namespace: pod.Namespace,
			ServiceAccount: identity.K8sServiceAccount{
				Namespace: pod.Namespace,
				Name:      pod.Spec.ServiceAccountName,
			},
			CreationTime: pod.GetCreationTimestamp().Time,
			WorkloadKind: workloadKind,
			WorkloadName: workloadName,
		}

		for idx := range pod.Spec.Containers {
			if pod.Spec.Containers[idx].ReadinessProbe != nil {
				p.Metadata.ReadinessProbes = append(p.Metadata.ReadinessProbes, pod.Spec.Containers[idx].ReadinessProbe)
			}
			if pod.Spec.Containers[idx].LivenessProbe != nil {
				p.Metadata.LivenessProbes = append(p.Metadata.LivenessProbes, pod.Spec.Containers[idx].LivenessProbe)
			}
			if pod.Spec.Containers[idx].StartupProbe != nil {
				p.Metadata.StartupProbes = append(p.Metadata.StartupProbes, pod.Spec.Containers[idx].StartupProbe)
			}
		}

		if len(pod.Status.PodIP) > 0 {
			p.Addr = pipy2.NewNetAddress(pod.Status.PodIP)
		}
	}

	// Verify Service account matches (cert to pod Service Account)
	if p.Identity.ToK8sServiceAccount() != p.Metadata.ServiceAccount {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMismatchedServiceAccount)).Str("proxy", p.String()).
			Msgf("Service Account referenced in NodeID (%s) does not match Service Account in Certificate (%s). This proxy is not allowed to join the mesh.", p.Metadata.ServiceAccount, p.Identity.ToK8sServiceAccount())
		return errServiceAccountMismatch
	}

	return nil
}

// recordVmMetadata records vm metadata and verifies the certificate issued for this vm
// is for the same service account as seen on the vm's service account
func (s *Server) recordVmMetadata(p *pipy2.Proxy) error {
	if p.Metadata == nil {
		vm, err := s.kubeController.GetVmForProxy(p)
		if err != nil {
			log.Warn().Str("proxy", p.String()).Msg("Could not find vm for connecting proxy. No metadata was recorded.")
			return nil
		}

		workloadKind := ""
		workloadName := ""
		for _, ref := range vm.GetOwnerReferences() {
			if ref.Controller != nil && *ref.Controller {
				workloadKind = ref.Kind
				workloadName = ref.Name
				break
			}
		}

		p.Metadata = &pipy2.ProxyMetadata{
			UID:       string(vm.UID),
			Name:      vm.Name,
			Namespace: vm.Namespace,
			ServiceAccount: identity.K8sServiceAccount{
				Namespace: vm.Namespace,
				Name:      vm.Spec.ServiceAccountName,
			},
			CreationTime: vm.GetCreationTimestamp().Time,
			WorkloadKind: workloadKind,
			WorkloadName: workloadName,
		}

		if vm.Spec.ReadinessProbe != nil {
			p.Metadata.ReadinessProbes = append(p.Metadata.ReadinessProbes, vm.Spec.ReadinessProbe)
		}
		if vm.Spec.LivenessProbe != nil {
			p.Metadata.LivenessProbes = append(p.Metadata.LivenessProbes, vm.Spec.LivenessProbe)
		}
		if vm.Spec.StartupProbe != nil {
			p.Metadata.StartupProbes = append(p.Metadata.StartupProbes, vm.Spec.StartupProbe)
		}

		if len(vm.Spec.MachineIP) > 0 {
			p.Addr = pipy2.NewNetAddress(vm.Spec.MachineIP)
		}
	}

	// Verify Service account matches (cert to pod Service Account)
	if p.Identity.ToK8sServiceAccount() != p.Metadata.ServiceAccount {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMismatchedServiceAccount)).Str("proxy", p.String()).
			Msgf("Service Account referenced in NodeID (%s) does not match Service Account in Certificate (%s). This proxy is not allowed to join the mesh.", p.Metadata.ServiceAccount, p.Identity.ToK8sServiceAccount())
		return errServiceAccountMismatch
	}

	return nil
}
