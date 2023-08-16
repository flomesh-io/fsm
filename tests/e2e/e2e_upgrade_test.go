package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	corev1 "k8s.io/api/core/v1"

	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Upgrade from latest",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 10,
	},
	func() {
		const ns = "upgrade-test"

		It("Tests upgrading the control plane", func() {
			Skip("Tests upgrading the control plane")

			if Td.InstType == NoInstall {
				Skip("test requires fresh FSM install")
			}

			if _, err := exec.LookPath("kubectl"); err != nil {
				Td.T.Fatal("\"kubectl\" command required and not found on PATH")
			}

			helmCfg := &action.Configuration{}
			Expect(helmCfg.Init(Td.Env.RESTClientGetter(), Td.FsmNamespace, "secret", Td.T.Logf)).To(Succeed())
			helmEnv := cli.New()

			// Install FSM with Helm vs. CLI so the test isn't dependent on
			// multiple versions of the CLI at once.
			Expect(Td.CreateNs(Td.FsmNamespace, nil)).To(Succeed())
			const releaseName = "fsm"
			i := action.NewInstall(helmCfg)

			i.ChartPathOptions.RepoURL = "https://flomesh-io.github.io/fsm"
			// On the main branch, this should refer to the latest release. On
			// release branches, it should specify the most recent patch of the
			// previous minor release. e.g. on the release-v1.0 branch, this
			// should be "0.11".
			i.Version = "1.1.0"
			i.Namespace = Td.FsmNamespace
			i.Wait = true
			i.ReleaseName = releaseName
			i.Timeout = 120 * time.Second
			vals := map[string]interface{}{
				"fsm": map[string]interface{}{
					"deployPrometheus": true,
					// Init container must be privileged if an OpenShift cluster is being used
					"enablePrivilegedInitContainer": Td.DeployOnOpenShift,

					// Reduce CPU so CI (capped at 2 CPU) can handle standing
					// up the new control plane before tearing the old one
					// down.
					"fsmController": map[string]interface{}{
						"resource": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu": "0.3",
							},
						},
					},
					"injector": map[string]interface{}{
						"resource": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu": "0.1",
							},
						},
					},
					"prometheus": map[string]interface{}{
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu":    "0.1",
								"memory": "256M",
							},
						},
					},
				},
			}

			chartPath, err := i.LocateChart("fsm", helmEnv)
			Expect(err).NotTo(HaveOccurred())
			ch, err := loader.Load(chartPath)
			Expect(err).NotTo(HaveOccurred())
			Td.T.Log("testing upgrade from chart version", ch.Metadata.Version)

			_, err = i.Run(ch, vals)
			Expect(err).NotTo(HaveOccurred())

			// Create Test NS
			Expect(Td.CreateNs(ns, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, ns)).To(Succeed())

			// Get simple pod definitions for the HTTP server
			serverSvcAccDef, serverPodDef, serverSvcDef, err := Td.GetOSSpecificHTTPBinPod("server", ns)
			Expect(err).NotTo(HaveOccurred())

			_, err = Td.CreateServiceAccount(ns, &serverSvcAccDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreatePod(ns, serverPodDef)
			Expect(err).NotTo(HaveOccurred())
			dstSvc, err := Td.CreateService(ns, serverSvcDef)
			Expect(err).NotTo(HaveOccurred())

			// Get simple Pod definitions for the client
			svcAccDef, srcPodDef, svcDef, err := Td.SimplePodApp(SimplePodAppDef{
				PodName:   "client",
				Namespace: ns,
				Command:   []string{"sleep", "365d"},
				Image:     "curlimages/curl",
				Ports:     []int{80},
				OS:        Td.ClusterOS,
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = Td.CreateServiceAccount(ns, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			srcPod, err := Td.CreatePod(ns, srcPodDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateService(ns, svcDef)
			Expect(err).NotTo(HaveOccurred())

			Expect(Td.WaitForPodsRunningReady(ns, 2, nil)).To(Succeed())

			// Deploy allow rule client->server
			httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
				SimpleAllowPolicy{
					RouteGroupName:    "routes",
					TrafficTargetName: "test-target",

					SourceNamespace:      ns,
					SourceSVCAccountName: svcAccDef.Name,

					DestinationNamespace:      ns,
					DestinationSvcAccountName: serverSvcAccDef.Name,
				},
			)
			_, err = Td.CreateHTTPRouteGroup(ns, httpRG)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateTrafficTarget(ns, trafficTarget)
			Expect(err).NotTo(HaveOccurred())

			// All ready. Expect client to reach server
			checkClientToServerOK := func() {
				By("Checking client can make requests to server")
				cond := Td.WaitForRepeatedSuccess(func() bool {
					result :=
						Td.HTTPRequest(HTTPRequestDef{
							SourceNs:        srcPod.Namespace,
							SourcePod:       srcPod.Name,
							SourceContainer: srcPod.Name,
							Destination:     fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace) + "/status/200",
						})

					if result.Err != nil || result.StatusCode != 200 {
						Td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
						return false
					}
					Td.T.Logf("> REST req succeeded: %d", result.StatusCode)
					return true
				}, 5 /*consecutive success threshold*/, 60*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())
			}

			checkProxiesConnected := func() {
				By("Checking all proxies are connected")
				prometheus, err := Td.GetFSMPrometheusHandle()
				Expect(err).NotTo(HaveOccurred())
				defer prometheus.Stop()
				cond := Td.WaitForRepeatedSuccess(func() bool {
					expectedProxyCount := float64(2)
					proxies, err := prometheus.VectorQuery("fsm_proxy_connect_count", time.Now())
					if err != nil {
						Td.T.Log("error querying prometheus:", err)
						return false
					}

					if proxies != expectedProxyCount {
						Td.T.Logf("expected query result %v, got %v", expectedProxyCount, proxies)
						return false
					}

					Td.T.Log("All proxies connected")
					return true
				}, 5 /*success threshold*/, 30*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())
			}

			checkProxiesConnected()
			checkClientToServerOK()

			By("Upgrading FSM")

			pullPolicy := corev1.PullAlways
			if Td.InstType == KindCluster {
				pullPolicy = corev1.PullIfNotPresent
				Expect(Td.LoadFSMImagesIntoKind()).To(Succeed())
			}

			setArgs := "--set=fsm.image.tag=" + Td.FsmImageTag + ",fsm.image.registry=" + Td.CtrRegistryServer + ",fsm.image.pullPolicy=" + string(pullPolicy) + ",fsm.deployPrometheus=true,fsm.enablePrivilegedInitContainer=" + strconv.FormatBool(Td.DeployOnOpenShift) + ",fsm.fsmController.resource.requests.cpu=0.3,fsm.injector.resource.requests.cpu=0.1,fsm.prometheus.resources.requests.cpu=0.1,fsm.prometheus.resources.requests.memory=256M"
			stdout, stderr, err := Td.RunLocal(filepath.FromSlash("../../bin/fsm"), "mesh", "upgrade", "--fsm-namespace="+Td.FsmNamespace, setArgs)
			Td.T.Log(stdout.String())
			if err != nil {
				Td.T.Log("stderr:\n" + stderr.String())
			}
			Expect(err).NotTo(HaveOccurred())

			// Verify that all the CRD's required by FSM are present in the cluster post an upgrade
			// TODO: Find a decent way to do this without relying on the kubectl binary
			// TODO: In the future when we bump the version on a CRD, we need to update this check to ensure that the version is the latest required version
			stdout, stderr, err = Td.RunLocal("kubectl", "get", "-f", filepath.FromSlash("../../cmd/fsm-bootstrap/crds"))
			Td.T.Log(stdout.String())
			if err != nil {
				Td.T.Log("stderr:\n" + stderr.String())
			}
			Expect(err).NotTo(HaveOccurred())

			checkClientToServerOK()
			checkProxiesConnected()
		})
	})
