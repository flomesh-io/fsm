package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flomesh-io/fsm/tests/framework"
	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("1 Client pod -> 1 Server pod test using cert-manager",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("CertManagerSimpleClientServer", func() {
			var (
				clientNamespace     = framework.RandomNameWithPrefix("client")
				serverNamespace     = framework.RandomNameWithPrefix("server")
				clientContainerName = framework.RandomNameWithPrefix("container")
				ns                  = []string{clientNamespace, serverNamespace}
			)

			It("Tests HTTP traffic for client pod -> server pod", func() {
				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.CertManager = "cert-manager"
				// Currently certs are rotated ~30-35s. This means we will rotate every other time we check, which is on a
				// 5 second period. We just add the 10 5 extra seconds to make sure the http requests succeed.
				installOpts.CertValidtyDuration = time.Second * 10
				installOpts.SetOverrides = []string{
					// increase timeout when using an external certificate provider due to
					// potential slowness issuing certs
					"fsm.injector.webhookTimeoutSeconds=30",
				}
				Expect(Td.InstallFSM(installOpts)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(Td.FsmNamespace, 5 /* 3 cert-manager pods, 1 controller, 1 injector */, nil)).To(Succeed())

				// Create namespaces
				for _, n := range ns {
					Expect(Td.CreateNs(n, nil)).To(Succeed())
					Expect(Td.AddNsToMesh(true, n)).To(Succeed())
				}

				// Get simple pod definitions for the HTTP server
				destinationPort := fortioHTTPPort
				serverSvcAccDef, serverPodDef, serverSvcDef, err := Td.SimplePodApp(
					SimplePodAppDef{
						PodName:   framework.RandomNameWithPrefix("pod"),
						Namespace: serverNamespace,
						Image:     fortioImageName,
						Ports:     []int{destinationPort},
						OS:        Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(serverNamespace, &serverSvcAccDef)
				Expect(err).NotTo(HaveOccurred())
				dstPod, err := Td.CreatePod(serverNamespace, serverPodDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(serverNamespace, serverSvcDef)
				Expect(err).NotTo(HaveOccurred())

				// Expect it to be up and running in it's receiver namespace
				Expect(Td.WaitForPodsRunningReady(serverNamespace, 1, nil)).To(Succeed())

				// Get simple Pod definitions for the client
				clientSvcAccDef, clientPodDef, clientSvcDef, err := Td.SimplePodApp(SimplePodAppDef{
					PodName:       framework.RandomNameWithPrefix("pod"),
					Namespace:     clientNamespace,
					ContainerName: clientContainerName,
					Image:         fortioImageName,
					Ports:         []int{destinationPort},
					OS:            Td.ClusterOS,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(clientNamespace, &clientSvcAccDef)
				Expect(err).NotTo(HaveOccurred())
				srcPod, err := Td.CreatePod(clientNamespace, clientPodDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(clientNamespace, clientSvcDef)
				Expect(err).NotTo(HaveOccurred())

				// Expect it to be up and running in it's receiver namespace
				Expect(Td.WaitForPodsRunningReady(clientNamespace, 1, nil)).To(Succeed())

				// Deploy allow rule client->server
				httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
					SimpleAllowPolicy{
						RouteGroupName:    "routes",
						TrafficTargetName: "target",

						SourceNamespace:      clientNamespace,
						SourceSVCAccountName: clientSvcAccDef.Name,

						DestinationNamespace:      serverNamespace,
						DestinationSvcAccountName: serverSvcAccDef.Name,
					})

				// Configs have to be put into a monitored NS, and fsm-system can't be by cli
				_, err = Td.CreateHTTPRouteGroup(serverNamespace, httpRG)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(serverNamespace, trafficTarget)
				Expect(err).NotTo(HaveOccurred())

				// All ready. Expect client to reach server
				// Need to get the pod though.
				for i := 0; i < 2; i++ {
					cond := Td.WaitForRepeatedSuccess(func() bool {
						result :=
							Td.FortioHTTPLoadTest(FortioHTTPLoadTestDef{
								HTTPRequestDef: HTTPRequestDef{
									SourceNs:        srcPod.Namespace,
									SourcePod:       srcPod.Name,
									SourceContainer: clientContainerName,

									Destination: fmt.Sprintf("%s.%s:%d", serverSvcDef.Name, dstPod.Namespace, destinationPort),
								},
							})

						if result.Err != nil || result.HasFailedHTTPRequests() {
							Td.T.Logf("> REST req has failed requests: %v", result.Err)
							return false
						}
						Td.T.Logf("> REST req succeeded. Status codes: %v", result.AllReturnCodes())
						return true
					}, 5 /*consecutive success threshold*/, 90*time.Second /*timeout*/)
					Expect(cond).To(BeTrue())
					time.Sleep(time.Second * 6) // 6 seconds guarantee the certs are rotated.
				}
			})
		})
	})
