package repo

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"sync/atomic"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/mitchellh/hashstructure/v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/ctok"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/injector"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
)

// PipyConfGeneratorJob is the job to generate pipy policy json
type PipyConfGeneratorJob struct {
	proxy      *pipy.Proxy
	repoServer *Server

	// Optional waiter
	done chan struct{}
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *PipyConfGeneratorJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *PipyConfGeneratorJob) Run() {
	defer close(job.done)
	if job.proxy == nil {
		return
	}

	if backlogs := atomic.LoadInt32(&job.proxy.Backlogs); backlogs > 0 {
		return
	}

	s := job.repoServer
	proxy := job.proxy

	atomic.AddInt32(&proxy.Backlogs, 1)
	proxy.Mutex.Lock()
	defer proxy.Mutex.Unlock()

	if backlogs := atomic.LoadInt32(&job.proxy.Backlogs); backlogs > 1 {
		atomic.AddInt32(&proxy.Backlogs, -1)
		return
	}

	atomic.AddInt32(&proxy.Backlogs, -1)

	proxyServices, err := s.proxyRegistry.ListProxyServices(proxy)
	if err != nil {
		log.Warn().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Msgf("Error looking up services for Sidecar with name=%s", proxy.GetName())
		return
	}

	if len(proxyServices) > 0 {
		sort.SliceStable(proxyServices, func(i, j int) bool {
			ps1 := proxyServices[i]
			ps2 := proxyServices[j]
			return ps1.Namespace < ps2.Namespace || ps1.Name < ps2.Name
		})
	}

	cataloger := s.catalog
	pipyConf := new(PipyConf)

	desiredSuffix := ""
	if proxy.Metadata != nil && len(proxy.Metadata.Namespace) > 0 {
		desiredSuffix = fmt.Sprintf("%s.svc.%s", proxy.Metadata.Namespace, service.GetTrustDomain())
		metrics, _ := injector.IsMetricsEnabled(s.kubeController, proxy.Metadata.Namespace)
		pipyConf.Metrics = metrics
	}

	start := time.Now()
	probes(proxy, pipyConf)
	features(s, proxy, pipyConf)
	certs(s, proxy, pipyConf, proxyServices)
	plugin(cataloger, s, pipyConf, proxy)
	inbound(cataloger, proxy.Identity, s, pipyConf, proxyServices, proxy)
	outbound(cataloger, proxy.Identity, s, pipyConf, proxy, s.cfg, desiredSuffix)
	egress(cataloger, proxy.Identity, s, pipyConf, proxy, desiredSuffix)
	forward(cataloger, proxy.Identity, s, pipyConf, proxy)
	cloudConnector(cataloger, pipyConf, s.cfg, proxy)
	balance(pipyConf)
	reorder(pipyConf)
	allowedEndpoints(pipyConf, s)
	dnsResolveDB(pipyConf, s.cfg)
	job.publishSidecarConf(s.repoClient, proxy, pipyConf)
	end := time.Now()

	log.Log().Str("proxy", proxy.GetCNPrefix()).
		Int("maxprocs", runtime.GOMAXPROCS(-1)).
		Str("elapsed", end.Sub(start).String()).
		Msg("Codebase Recalculated")
}

func allowedEndpoints(pipyConf *PipyConf, s *Server) {
	ready := pipyConf.copyAllowedEndpoints(s.kubeController, s.proxyRegistry)
	if !ready {
		if s.retryProxiesJob != nil {
			s.retryProxiesJob()
		}
	}
}

func balance(pipyConf *PipyConf) {
	pipyConf.rebalancedOutboundClusters()
	pipyConf.rebalancedForwardClusters()
}

func reorder(pipyConf *PipyConf) {
	if pipyConf.Outbound != nil && pipyConf.Outbound.TrafficMatches != nil {
		for _, trafficMatches := range pipyConf.Outbound.TrafficMatches {
			for _, trafficMatch := range trafficMatches {
				for _, routeRules := range trafficMatch.HTTPServiceRouteRules {
					routeRules.RouteRules.sort()
				}
			}
		}
		pipyConf.Outbound.TrafficMatches.Sort()
	}

	if pipyConf.Inbound != nil && pipyConf.Inbound.TrafficMatches != nil {
		for _, trafficMatches := range pipyConf.Inbound.TrafficMatches {
			for _, routeRules := range trafficMatches.HTTPServiceRouteRules {
				routeRules.sort()
			}
		}
	}
}

func egress(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, proxy *pipy.Proxy, desiredSuffix string) bool {
	egressTrafficPolicy, egressErr := cataloger.GetEgressTrafficPolicy(serviceIdentity)
	if egressErr != nil {
		if s.retryProxiesJob != nil {
			s.retryProxiesJob()
		}
		return false
	}

	if egressTrafficPolicy != nil {
		egressDependClusters := generatePipyEgressTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf,
			egressTrafficPolicy, desiredSuffix)
		if len(egressDependClusters) > 0 {
			if ready := generatePipyEgressTrafficBalancePolicy(cataloger, proxy, serviceIdentity, pipyConf,
				egressTrafficPolicy, egressDependClusters); !ready {
				if s.retryProxiesJob != nil {
					s.retryProxiesJob()
				}
				return false
			}
		}
	}
	return true
}

func forward(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, _ *pipy.Proxy) bool {
	egressGatewayPolicy, egressErr := cataloger.GetEgressGatewayPolicy()
	if egressErr != nil {
		if s.retryProxiesJob != nil {
			s.retryProxiesJob()
		}
		return false
	}
	if egressGatewayPolicy != nil {
		if ready := generatePipyEgressTrafficForwardPolicy(cataloger, serviceIdentity, pipyConf,
			egressGatewayPolicy); !ready {
			if s.retryProxiesJob != nil {
				s.retryProxiesJob()
			}
			return false
		}
	}
	return true
}

func outbound(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, proxy *pipy.Proxy, cfg configurator.Configurator, desiredSuffix string) bool {
	outboundTrafficPolicy := cataloger.GetOutboundMeshTrafficPolicy(serviceIdentity)
	if cfg.IsLocalDNSProxyEnabled() && !cfg.IsWildcardDNSProxyEnabled() {
		if len(outboundTrafficPolicy.ServicesResolvableSet) > 0 {
			if pipyConf.DNSResolveDB == nil {
				pipyConf.DNSResolveDB = make(map[string][]interface{})
			}
			for k, v := range outboundTrafficPolicy.ServicesResolvableSet {
				pipyConf.DNSResolveDB[k] = v
			}
		}
	}
	outboundDependClusters := generatePipyOutboundTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf,
		cfg, outboundTrafficPolicy, desiredSuffix)
	if len(outboundDependClusters) > 0 {
		if ready := generatePipyOutboundTrafficBalancePolicy(cataloger, cfg, proxy, serviceIdentity, pipyConf,
			outboundTrafficPolicy, outboundDependClusters); !ready {
			if s.retryProxiesJob != nil {
				s.retryProxiesJob()
			}
			return false
		}
	}
	return true
}

func inbound(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, proxyServices []service.MeshService, proxy *pipy.Proxy) {
	// Build inbound mesh route configurations. These route configurations allow
	// the services associated with this proxy to accept traffic from downstream
	// clients on allowed routes.
	inboundTrafficPolicy := cataloger.GetInboundMeshTrafficPolicy(serviceIdentity, proxyServices)
	generatePipyInboundTrafficPolicy(cataloger, serviceIdentity, pipyConf, inboundTrafficPolicy, s.certManager.GetTrustDomain(), proxy)
	if len(proxyServices) > 0 {
		for _, svc := range proxyServices {
			if ingressTrafficPolicy, ingressErr := cataloger.GetIngressTrafficPolicy(svc); ingressErr == nil {
				if ingressTrafficPolicy != nil {
					generatePipyIngressTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf, ingressTrafficPolicy)
				}
			}
			if aclTrafficPolicy, aclErr := cataloger.GetAccessControlTrafficPolicy(svc); aclErr == nil {
				if aclTrafficPolicy != nil {
					generatePipyAccessControlTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf, aclTrafficPolicy)
				}
			}
			if expTrafficPolicy, expErr := cataloger.GetExportTrafficPolicy(svc); expErr == nil {
				if expTrafficPolicy != nil {
					generatePipyServiceExportTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf, expTrafficPolicy)
				}
			}
		}
	}
}

func plugin(cataloger catalog.MeshCataloger, s *Server, pipyConf *PipyConf, proxy *pipy.Proxy) {
	pipyConf.Chains = nil

	defer func() {
		if pipyConf.Chains == nil {
			setSidecarChain(s.cfg, pipyConf, nil, nil)
		}
	}()

	if !s.cfg.GetFeatureFlags().EnablePluginPolicy {
		return
	}

	pluginChains := cataloger.GetPluginChains()
	if len(pluginChains) == 0 {
		return
	}

	var labelMap map[string]string
	var ns *corev1.Namespace

	if proxy.VM {
		vm, err := s.kubeController.GetVmForProxy(proxy)
		if err != nil {
			log.Warn().Str("proxy", proxy.String()).Msg("Could not find VM for connecting proxy.")
			return
		}

		ns = s.kubeController.GetNamespace(vm.Namespace)
		if ns == nil {
			log.Warn().Str("proxy", proxy.String()).Str("namespace", vm.Namespace).Msg("Could not find namespace for connecting proxy.")
		}

		labelMap = vm.Labels
	} else {
		pod, err := s.kubeController.GetPodForProxy(proxy)
		if err != nil {
			log.Warn().Str("proxy", proxy.String()).Msg("Could not find pod for connecting proxy.")
			return
		}

		ns = s.kubeController.GetNamespace(pod.Namespace)
		if ns == nil {
			log.Warn().Str("proxy", proxy.String()).Str("namespace", pod.Namespace).Msg("Could not find namespace for connecting proxy.")
		}

		labelMap = pod.Labels
	}

	pluginSet, pluginPri := s.updatePlugins()
	plugin2MountPoint2Config, mountPoint2Plugins := walkPluginChain(pluginChains, ns, labelMap, pluginSet, s, proxy)
	meshSvc2Plugin2MountPoint2Config := walkPluginConfig(cataloger, plugin2MountPoint2Config)

	pipyConf.pluginPolicies = meshSvc2Plugin2MountPoint2Config
	setSidecarChain(s.cfg, pipyConf, pluginPri, mountPoint2Plugins)

	pipyConf.PluginSetV = s.pluginSetVersion
}

func certs(s *Server, proxy *pipy.Proxy, pipyConf *PipyConf, proxyServices []service.MeshService) {
	if mc, ok := s.catalog.(*catalog.MeshCatalog); ok {
		meshConf := mc.GetConfigurator()
		if !(*meshConf).GetSidecarDisabledMTLS() {
			cnPrefix := proxy.Identity.String()
			if proxy.SidecarCert == nil {
				pipyConf.Certificate = nil
				sidecarCert := s.certManager.GetCertificate(cnPrefix)
				if sidecarCert == nil {
					proxy.SidecarCert = nil
				} else {
					proxy.SidecarCert = sidecarCert
				}
			}
			if proxy.SidecarCert == nil || s.certManager.ShouldRotate(proxy.SidecarCert) {
				pipyConf.Certificate = nil
				now := time.Now()
				certValidityPeriod := s.cfg.GetServiceCertValidityPeriod()
				certExpiration := now.Add(certValidityPeriod)
				certValidityPeriod = certExpiration.Sub(now)

				var sans []string
				if len(proxyServices) > 0 {
					san := (*meshConf).GetServiceAccessNames()
					for _, proxySvc := range proxyServices {
						sans = append(sans, k8s.GetHostnamesForService(proxySvc, san, true)...)
					}
				}
				for {
					sidecarCert, certErr := s.certManager.IssueCertificate(cnPrefix, certificate.Service,
						certificate.SubjectAlternativeNames(sans...),
						certificate.ValidityDurationProvided(&certValidityPeriod))
					if certErr != nil {
						log.Err(certErr).Msgf("error IssueCertificate for cnPrefix:%s", cnPrefix)
					} else if !s.certManager.ShouldRotate(sidecarCert) {
						proxy.SidecarCert = sidecarCert
						break
					}
					time.Sleep(time.Second * 5)
				}
			}
		} else {
			proxy.SidecarCert = nil
		}
	}
}

func features(s *Server, proxy *pipy.Proxy, pipyConf *PipyConf) {
	if mc, ok := s.catalog.(*catalog.MeshCatalog); ok {
		meshConf := mc.GetConfigurator()
		proxy.MeshConf = meshConf
		pipyConf.setSidecarLogLevel((*meshConf).GetMeshConfig().Spec.Sidecar.LogLevel)
		pipyConf.setSidecarTimeout((*meshConf).GetMeshConfig().Spec.Sidecar.SidecarTimeout)
		pipyConf.setEnableSidecarActiveHealthChecks((*meshConf).GetFeatureFlags().EnableSidecarActiveHealthChecks)
		pipyConf.setEnableAutoDefaultRoute((*meshConf).GetFeatureFlags().EnableAutoDefaultRoute)
		pipyConf.setEnableEgress((*meshConf).IsEgressEnabled())
		pipyConf.setHTTP1PerRequestLoadBalancing((*meshConf).GetMeshConfig().Spec.Traffic.HTTP1PerRequestLoadBalancing)
		pipyConf.setHTTP2PerRequestLoadBalancing((*meshConf).GetMeshConfig().Spec.Traffic.HTTP2PerRequestLoadBalancing)
		pipyConf.setEnablePermissiveTrafficPolicyMode((*meshConf).IsPermissiveTrafficPolicyMode())
		pipyConf.setLocalDNSProxy((*meshConf).IsLocalDNSProxyEnabled(), meshConf)
		pipyConf.setObservabilityTracing((*meshConf).IsTracingEnabled(), meshConf)
		pipyConf.setObservabilityRemoteLogging((*meshConf).IsRemoteLoggingEnabled(), meshConf)
		clusterProps := (*meshConf).GetMeshConfig().Spec.ClusterSet.Properties
		if len(clusterProps) > 0 {
			pipyConf.Spec.ClusterSet = make(map[string]string)
			for _, prop := range clusterProps {
				pipyConf.Spec.ClusterSet[prop.Name] = prop.Value
			}
		}
	}
}

func probes(proxy *pipy.Proxy, pipyConf *PipyConf) {
	if proxy.Metadata != nil {
		if len(proxy.Metadata.StartupProbes) > 0 {
			for idx := range proxy.Metadata.StartupProbes {
				pipyConf.Spec.Probes.StartupProbes = append(pipyConf.Spec.Probes.StartupProbes, *proxy.Metadata.StartupProbes[idx])
			}
		}
		if len(proxy.Metadata.LivenessProbes) > 0 {
			for idx := range proxy.Metadata.LivenessProbes {
				pipyConf.Spec.Probes.LivenessProbes = append(pipyConf.Spec.Probes.LivenessProbes, *proxy.Metadata.LivenessProbes[idx])
			}
		}
		if len(proxy.Metadata.ReadinessProbes) > 0 {
			for idx := range proxy.Metadata.ReadinessProbes {
				pipyConf.Spec.Probes.ReadinessProbes = append(pipyConf.Spec.Probes.ReadinessProbes, *proxy.Metadata.ReadinessProbes[idx])
			}
		}
	}
}

func cloudConnector(cataloger catalog.MeshCataloger, pipyConf *PipyConf, cfg configurator.Configurator, proxy *pipy.Proxy) {
	if !cfg.IsLocalDNSProxyEnabled() || cfg.IsWildcardDNSProxyEnabled() {
		return
	}
	if proxy.Metadata == nil {
		return
	}
	if len(proxy.Metadata.Namespace) == 0 {
		return
	}
	kubeController := cataloger.GetKubeController()
	svcList := kubeController.ListServices()
	for _, svc := range svcList {
		ns := kubeController.GetNamespace(svc.Namespace)
		if !ctok.IsSyncCloudNamespace(ns) {
			continue
		}
		if len(svc.Annotations) > 0 {
			if pipyConf.DNSResolveDB == nil {
				pipyConf.DNSResolveDB = make(map[string][]interface{})
			}
			resolvableIPSet := mapset.NewSet()
			if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
				svcMeta := connector.Decode(svc, v)
				for addr := range svcMeta.Endpoints {
					resolvableIPSet.Add(string(addr))
				}
			}
			if resolvableIPSet.Cardinality() > 0 {
				if addrs, exists := pipyConf.DNSResolveDB[svc.Name]; exists {
					for _, addr := range addrs {
						if !resolvableIPSet.Contains(addr) {
							resolvableIPSet.Add(addr)
						}
					}
				}
				addrItems := resolvableIPSet.ToSlice()
				sort.SliceStable(addrItems, func(i, j int) bool {
					addr1 := addrItems[i].(string)
					addr2 := addrItems[j].(string)
					return addr1 < addr2
				})
				pipyConf.DNSResolveDB[fmt.Sprintf("%s.%s.svc.%s", svc.Name, proxy.Metadata.Namespace, service.GetTrustDomain())] = addrItems
				delete(pipyConf.DNSResolveDB, svc.Name)
			}
		}
	}
}

func dnsResolveDB(pipyConf *PipyConf, cfg configurator.Configurator) {
	if !cfg.IsLocalDNSProxyEnabled() {
		return
	}
	if pipyConf.DNSResolveDB == nil {
		pipyConf.DNSResolveDB = make(map[string][]interface{})
	}
	dnsProxy := cfg.GetMeshConfig().Spec.Sidecar.LocalDNSProxy
	if cfg.IsWildcardDNSProxyEnabled() {
		ipv4s := make([]interface{}, 0)
		for _, ipv4 := range dnsProxy.Wildcard.IPv4 {
			ipv4s = append(ipv4s, ipv4)
		}
		pipyConf.DNSResolveDB["*"] = ipv4s
	} else {
		if len(dnsProxy.DB) > 0 {
			for _, db := range dnsProxy.DB {
				if len(db.IPv4) > 0 {
					ipv4s := make([]interface{}, 0)
					for _, ipv4 := range db.IPv4 {
						ipv4s = append(ipv4s, ipv4)
					}
					pipyConf.DNSResolveDB[db.DN] = ipv4s
				}
			}
		}
	}
}

func (job *PipyConfGeneratorJob) publishSidecarConf(repoClient *client.PipyRepoClient, proxy *pipy.Proxy, pipyConf *PipyConf) {
	pipyConf.Ts = nil
	pipyConf.Version = nil
	pipyConf.Certificate = nil
	if proxy.SidecarCert != nil {
		pipyConf.Certificate = &Certificate{
			Expiration: proxy.SidecarCert.Expiration.Format("2006-01-02 15:04:05"),
			CommonName: &proxy.SidecarCert.CommonName,
			CertChain:  string(proxy.SidecarCert.CertChain),
			PrivateKey: string(proxy.SidecarCert.PrivateKey),
			IssuingCA:  string(proxy.SidecarCert.IssuingCA),
		}
	}

	if !prettyConfig() {
		pipyConf.Pack()
	}

	codebaseCurV, err := hashstructure.Hash(pipyConf, hashstructure.FormatV2,
		&hashstructure.HashOptions{
			ZeroNil:         true,
			IgnoreZeroValue: true,
			SlicesAsSets:    true,
		})

	if err == nil {
		codebasePreV := proxy.ETag
		if codebaseCurV != codebasePreV {
			codebase := fmt.Sprintf("%s/%s", fsmSidecarCodebase, proxy.GetCNPrefix())
			success, err := repoClient.DeriveCodebase(codebase, fsmCodebaseRepo, codebaseCurV-2)
			if success {
				ts := time.Now()
				pipyConf.Ts = &ts
				version := fmt.Sprintf("%d", codebaseCurV)
				pipyConf.Version = &version

				var bytes []byte
				if prettyConfig() {
					bytes, _ = json.MarshalIndent(pipyConf, "", " ")
				} else {
					bytes, _ = json.Marshal(pipyConf)
				}
				_, err = repoClient.Batch(fmt.Sprintf("%d", codebaseCurV-1), []client.Batch{
					{
						Basepath: codebase,
						Items: []client.BatchItem{
							{
								Filename: fsmCodebaseConfig,
								Content:  bytes,
							},
						},
					},
				})
			}
			if err != nil || !success {
				if err != nil {
					log.Error().Err(err)
				}
				_, _ = repoClient.Delete(codebase)
			} else {
				proxy.ETag = codebaseCurV
				log.Log().Str("proxy", proxy.GetCNPrefix()).
					Str("id", fmt.Sprintf("%05d", proxy.ID)).
					Str("prev", fmt.Sprintf("%020d", codebasePreV)).
					Str("curv", fmt.Sprintf("%020d", codebaseCurV)).
					Msg("Codebase Regenerated.")
			}
		}
	}
}

// JobName implementation for this job, for logging purposes
func (job *PipyConfGeneratorJob) JobName() string {
	return fmt.Sprintf("pipyJob-%s", job.proxy.GetName())
}
